"""Speaker diarization service using pyannote.audio."""

import asyncio
import os
import tempfile
from pathlib import Path
from typing import List, Optional, Callable, Tuple
from dataclasses import dataclass, field
import logging

import numpy as np
import soundfile as sf
from pydub import AudioSegment
from scipy import signal

logger = logging.getLogger(__name__)

# Constants from SPEC.md
MIN_SEGMENT_DURATION_MS = 1500  # 1.5s minimum for embedding extraction
OVERLAP_THRESHOLD_MS = 500  # Overlapping speech > 500ms marked as "Multiple Speakers"
BANDPASS_LOW_HZ = 300  # Band-pass filter low cutoff
BANDPASS_HIGH_HZ = 3400  # Band-pass filter high cutoff


@dataclass
class DiarizationSegment:
    """A segment with speaker information."""
    speaker_id: str  # Temporary ID like "SPEAKER_00", "SPEAKER_01"
    start_ms: int
    end_ms: int
    has_overlap: bool = False  # True if overlapping speech detected (> 500ms)


@dataclass
class DiarizationResult:
    """Complete diarization result."""
    segments: List[DiarizationSegment]
    num_speakers: int
    speaker_ids: List[str]
    overlap_regions: List[Tuple[int, int]] = field(default_factory=list)  # (start_ms, end_ms)


def apply_bandpass_filter(
    audio: np.ndarray,
    sample_rate: int,
    low_hz: int = BANDPASS_LOW_HZ,
    high_hz: int = BANDPASS_HIGH_HZ,
) -> np.ndarray:
    """
    Apply band-pass filter to audio to remove hum/hiss.

    Per SPEC.md Section 4: Preprocessing with band-pass filter 300Hz - 3400Hz
    to remove low-frequency hum and high-frequency hiss.

    Args:
        audio: Audio waveform as numpy array
        sample_rate: Sample rate in Hz
        low_hz: Low cutoff frequency (default 300Hz)
        high_hz: High cutoff frequency (default 3400Hz)

    Returns:
        Filtered audio
    """
    nyquist = sample_rate / 2
    low = low_hz / nyquist
    high = high_hz / nyquist

    # Ensure frequencies are in valid range
    low = max(0.001, min(low, 0.99))
    high = max(low + 0.001, min(high, 0.99))

    # Design Butterworth bandpass filter (4th order)
    b, a = signal.butter(4, [low, high], btype='band')

    # Apply filter with zero-phase (no delay)
    filtered = signal.filtfilt(b, a, audio)

    return filtered.astype(audio.dtype)


def detect_overlaps(
    segments: List[DiarizationSegment],
    threshold_ms: int = OVERLAP_THRESHOLD_MS,
) -> List[Tuple[int, int]]:
    """
    Detect regions where speakers overlap for > threshold_ms.

    Per SPEC.md Section 3.2: Overlapping speech > 500ms must be flagged
    as "Multiple Speakers" rather than forcing a wrong single-speaker label.

    Args:
        segments: List of diarization segments
        threshold_ms: Minimum overlap duration to flag (default 500ms)

    Returns:
        List of (start_ms, end_ms) for overlap regions
    """
    if len(segments) < 2:
        return []

    # Sort by start time
    sorted_segs = sorted(segments, key=lambda s: s.start_ms)

    overlaps = []

    for i in range(len(sorted_segs)):
        for j in range(i + 1, len(sorted_segs)):
            seg_a = sorted_segs[i]
            seg_b = sorted_segs[j]

            # Check for overlap
            overlap_start = max(seg_a.start_ms, seg_b.start_ms)
            overlap_end = min(seg_a.end_ms, seg_b.end_ms)
            overlap_duration = overlap_end - overlap_start

            if overlap_duration >= threshold_ms:
                overlaps.append((overlap_start, overlap_end))

    # Merge adjacent overlaps
    if not overlaps:
        return []

    overlaps.sort(key=lambda x: x[0])
    merged = [overlaps[0]]

    for start, end in overlaps[1:]:
        if start <= merged[-1][1]:
            merged[-1] = (merged[-1][0], max(merged[-1][1], end))
        else:
            merged.append((start, end))

    return merged


def mark_overlapping_segments(
    segments: List[DiarizationSegment],
    overlap_regions: List[Tuple[int, int]],
) -> None:
    """
    Mark segments that fall within overlap regions.

    Args:
        segments: Segments to check (modified in place)
        overlap_regions: List of (start_ms, end_ms) overlap regions
    """
    for seg in segments:
        for overlap_start, overlap_end in overlap_regions:
            # Check if segment significantly overlaps with an overlap region
            overlap_with_region_start = max(seg.start_ms, overlap_start)
            overlap_with_region_end = min(seg.end_ms, overlap_end)
            overlap_duration = overlap_with_region_end - overlap_with_region_start

            if overlap_duration >= OVERLAP_THRESHOLD_MS:
                seg.has_overlap = True
                break


class DiarizationService:
    """Service for speaker diarization using pyannote.audio."""

    def __init__(
        self,
        hf_token: Optional[str] = None,
        num_speakers: Optional[int] = None,
        min_speakers: int = 1,
        max_speakers: int = 10,
        apply_preprocessing: bool = True,
    ):
        """
        Initialize diarization service.

        Args:
            hf_token: Hugging Face token (required for pyannote models)
            num_speakers: Fixed number of speakers (if known)
            min_speakers: Minimum expected speakers
            max_speakers: Maximum expected speakers
            apply_preprocessing: Apply band-pass filter preprocessing
        """
        self.hf_token = hf_token or os.getenv("HF_TOKEN")
        self.num_speakers = num_speakers
        self.min_speakers = min_speakers
        self.max_speakers = max_speakers
        self.apply_preprocessing = apply_preprocessing
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

            # Detect overlapping speech regions
            overlap_regions = detect_overlaps(segments)

            # Mark segments that fall in overlap regions
            mark_overlapping_segments(segments, overlap_regions)

            # Convert to readable speaker IDs (Speaker A, B, C, etc.)
            speaker_map = {}
            for i, sid in enumerate(sorted(speaker_ids)):
                speaker_map[sid] = f"Speaker {chr(65 + i)}"  # A, B, C, ...

            for seg in segments:
                seg.speaker_id = speaker_map.get(seg.speaker_id, seg.speaker_id)

            if on_progress:
                on_progress(95, "Finalizing results...")

            return DiarizationResult(
                segments=segments,
                num_speakers=len(speaker_ids),
                speaker_ids=list(speaker_map.values()),
                overlap_regions=overlap_regions,
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
        """
        Convert audio to WAV format and apply preprocessing if enabled.

        Per SPEC.md Section 4: Preprocessing with band-pass filter 300Hz - 3400Hz
        """
        path = Path(audio_path)
        needs_conversion = path.suffix.lower() != ".wav"
        needs_preprocessing = self.apply_preprocessing

        if not needs_conversion and not needs_preprocessing:
            return audio_path

        if on_progress:
            on_progress(15, "Converting audio format...")

        def convert_and_preprocess():
            # Load audio
            if needs_conversion:
                audio = AudioSegment.from_file(audio_path)
                # Convert to 16kHz mono
                audio = audio.set_frame_rate(16000).set_channels(1)
                temp_path = tempfile.mktemp(suffix=".wav")
                audio.export(temp_path, format="wav")
                working_path = temp_path
            else:
                working_path = audio_path

            if needs_preprocessing:
                # Apply band-pass filter
                if on_progress:
                    on_progress(18, "Applying audio preprocessing...")

                # Load as numpy array
                audio_data, sr = sf.read(working_path)

                # Ensure mono
                if len(audio_data.shape) > 1:
                    audio_data = audio_data.mean(axis=1)

                # Apply band-pass filter (300Hz - 3400Hz)
                filtered_audio = apply_bandpass_filter(audio_data, sr)

                # Save to temp file
                preprocessed_path = tempfile.mktemp(suffix=".wav")
                sf.write(preprocessed_path, filtered_audio, sr)

                # Clean up intermediate file if we created one
                if needs_conversion and working_path != audio_path:
                    os.remove(working_path)

                return preprocessed_path

            return working_path

        wav_path = await asyncio.get_event_loop().run_in_executor(None, convert_and_preprocess)
        return wav_path

    async def extract_speaker_audio(
        self,
        audio_path: str,
        segments: List[DiarizationSegment],
        speaker_id: str,
        min_duration_ms: int = MIN_SEGMENT_DURATION_MS,
        exclude_overlaps: bool = True,
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

                # Skip overlapping segments (per SPEC.md: never extract embeddings from overlaps)
                if exclude_overlaps and seg.has_overlap:
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
