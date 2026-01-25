"""Speaker diarization service using pyannote.audio."""

import asyncio
import os
import tempfile
from pathlib import Path
from typing import List, Optional, Callable, Tuple
from dataclasses import dataclass
import logging

import numpy as np
import soundfile as sf
from pydub import AudioSegment

logger = logging.getLogger(__name__)


@dataclass
class DiarizationSegment:
    """A segment with speaker information."""
    speaker_id: str  # Temporary ID like "SPEAKER_00", "SPEAKER_01"
    start_ms: int
    end_ms: int


@dataclass
class DiarizationResult:
    """Complete diarization result."""
    segments: List[DiarizationSegment]
    num_speakers: int
    speaker_ids: List[str]


class DiarizationService:
    """Service for speaker diarization using pyannote.audio."""

    def __init__(
        self,
        hf_token: Optional[str] = None,
        num_speakers: Optional[int] = None,
        min_speakers: int = 1,
        max_speakers: int = 10,
    ):
        """
        Initialize diarization service.

        Args:
            hf_token: Hugging Face token (required for pyannote models)
            num_speakers: Fixed number of speakers (if known)
            min_speakers: Minimum expected speakers
            max_speakers: Maximum expected speakers
        """
        self.hf_token = hf_token or os.getenv("HF_TOKEN")
        self.num_speakers = num_speakers
        self.min_speakers = min_speakers
        self.max_speakers = max_speakers
        self._pipeline = None

    async def _load_pipeline(self):
        """Lazy load the diarization pipeline."""
        if self._pipeline is not None:
            return

        def load():
            try:
                from pyannote.audio import Pipeline
                import torch

                # Determine device
                if torch.backends.mps.is_available():
                    device = torch.device("mps")
                elif torch.cuda.is_available():
                    device = torch.device("cuda")
                else:
                    device = torch.device("cpu")

                logger.info(f"Loading pyannote pipeline on {device}")

                pipeline = Pipeline.from_pretrained(
                    "pyannote/speaker-diarization-3.1",
                    use_auth_token=self.hf_token,
                )
                pipeline.to(device)
                return pipeline
            except Exception as e:
                logger.error(f"Failed to load pyannote pipeline: {e}")
                raise

        self._pipeline = await asyncio.get_event_loop().run_in_executor(None, load)

    async def diarize(
        self,
        audio_path: str,
        on_progress: Optional[Callable[[float, str], None]] = None,
    ) -> DiarizationResult:
        """
        Perform speaker diarization on an audio file.

        Args:
            audio_path: Path to the audio file
            on_progress: Optional callback for progress updates

        Returns:
            DiarizationResult with speaker segments
        """
        # Ensure pipeline is loaded
        if on_progress:
            on_progress(10, "Loading diarization model...")

        await self._load_pipeline()

        # Convert to WAV if needed
        wav_path = await self._ensure_wav(audio_path, on_progress)

        try:
            if on_progress:
                on_progress(30, "Analyzing speakers...")

            # Run diarization
            def run_diarization():
                params = {}
                if self.num_speakers:
                    params["num_speakers"] = self.num_speakers
                else:
                    params["min_speakers"] = self.min_speakers
                    params["max_speakers"] = self.max_speakers

                return self._pipeline(wav_path, **params)

            diarization = await asyncio.get_event_loop().run_in_executor(
                None, run_diarization
            )

            if on_progress:
                on_progress(90, "Processing speaker segments...")

            # Extract segments
            segments = []
            speaker_ids = set()

            for turn, _, speaker in diarization.itertracks(yield_label=True):
                speaker_id = speaker.replace("SPEAKER_", "Speaker ")
                speaker_ids.add(speaker_id)

                segments.append(DiarizationSegment(
                    speaker_id=speaker_id,
                    start_ms=int(turn.start * 1000),
                    end_ms=int(turn.end * 1000),
                ))

            # Sort by start time
            segments.sort(key=lambda s: s.start_ms)

            # Convert to readable speaker IDs (Speaker A, B, C, etc.)
            speaker_map = {}
            for i, sid in enumerate(sorted(speaker_ids)):
                speaker_map[sid] = f"Speaker {chr(65 + i)}"  # A, B, C, ...

            for seg in segments:
                seg.speaker_id = speaker_map.get(seg.speaker_id, seg.speaker_id)

            return DiarizationResult(
                segments=segments,
                num_speakers=len(speaker_ids),
                speaker_ids=list(speaker_map.values()),
            )

        finally:
            # Clean up temp file if we created one
            if wav_path != audio_path and os.path.exists(wav_path):
                os.remove(wav_path)

    async def _ensure_wav(
        self,
        audio_path: str,
        on_progress: Optional[Callable[[float, str], None]] = None,
    ) -> str:
        """Convert audio to WAV format if needed."""
        path = Path(audio_path)

        if path.suffix.lower() == ".wav":
            return audio_path

        if on_progress:
            on_progress(20, "Converting audio format...")

        def convert():
            audio = AudioSegment.from_file(audio_path)
            # Convert to 16kHz mono
            audio = audio.set_frame_rate(16000).set_channels(1)

            temp_path = tempfile.mktemp(suffix=".wav")
            audio.export(temp_path, format="wav")
            return temp_path

        wav_path = await asyncio.get_event_loop().run_in_executor(None, convert)
        return wav_path

    async def extract_speaker_audio(
        self,
        audio_path: str,
        segments: List[DiarizationSegment],
        speaker_id: str,
        min_duration_ms: int = 1000,
    ) -> List[Tuple[np.ndarray, int, int]]:
        """
        Extract audio segments for a specific speaker.

        Args:
            audio_path: Path to the audio file
            segments: Diarization segments
            speaker_id: The speaker to extract
            min_duration_ms: Minimum segment duration to include

        Returns:
            List of (audio_data, start_ms, end_ms) tuples
        """
        def extract():
            audio, sr = sf.read(audio_path)
            if len(audio.shape) > 1:
                audio = audio.mean(axis=1)  # Convert to mono

            results = []
            for seg in segments:
                if seg.speaker_id != speaker_id:
                    continue

                duration = seg.end_ms - seg.start_ms
                if duration < min_duration_ms:
                    continue

                start_sample = int(seg.start_ms * sr / 1000)
                end_sample = int(seg.end_ms * sr / 1000)
                segment_audio = audio[start_sample:end_sample]

                results.append((segment_audio, seg.start_ms, seg.end_ms))

            return results

        return await asyncio.get_event_loop().run_in_executor(None, extract)


class SimpleDiarizationService:
    """
    Fallback diarization using simple energy-based segmentation.
    Used when pyannote is not available.
    """

    async def diarize(
        self,
        audio_path: str,
        on_progress: Optional[Callable[[float, str], None]] = None,
    ) -> DiarizationResult:
        """Simple VAD-based segmentation without speaker separation."""
        if on_progress:
            on_progress(30, "Analyzing audio segments...")

        def segment():
            import librosa

            y, sr = librosa.load(audio_path, sr=16000, mono=True)

            # Use librosa's onset detection for basic segmentation
            onset_frames = librosa.onset.onset_detect(
                y=y, sr=sr, units="frames",
                hop_length=512,
                backtrack=True,
            )

            # Convert to time
            onset_times = librosa.frames_to_time(onset_frames, sr=sr, hop_length=512)

            # Create segments between onsets
            segments = []
            for i in range(len(onset_times)):
                start = onset_times[i]
                end = onset_times[i + 1] if i + 1 < len(onset_times) else len(y) / sr

                # Skip very short segments
                if (end - start) < 0.5:
                    continue

                segments.append(DiarizationSegment(
                    speaker_id="Speaker A",  # Single speaker fallback
                    start_ms=int(start * 1000),
                    end_ms=int(end * 1000),
                ))

            return segments

        segments = await asyncio.get_event_loop().run_in_executor(None, segment)

        return DiarizationResult(
            segments=segments,
            num_speakers=1,
            speaker_ids=["Speaker A"],
        )
