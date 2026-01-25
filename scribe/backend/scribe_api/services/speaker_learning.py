"""Speaker learning service for voice identification."""

import asyncio
import os
from typing import List, Optional, Tuple
from dataclasses import dataclass, field
from datetime import datetime
import logging
import struct

import numpy as np

logger = logging.getLogger(__name__)

# Constants
EMBEDDING_DIM = 192  # ECAPA-TDNN embedding dimension
HIGH_CONFIDENCE_THRESHOLD = 0.75
SUGGESTION_THRESHOLD = 0.60
MAX_SAMPLES_PER_SPEAKER = 50
ADAPTATION_RATE = 0.95  # Slow adaptation to prevent drift


def embedding_to_bytes(embedding: np.ndarray) -> bytes:
    """Convert numpy embedding to bytes for storage."""
    return embedding.astype(np.float32).tobytes()


def bytes_to_embedding(data: bytes) -> np.ndarray:
    """Convert stored bytes back to numpy embedding."""
    return np.frombuffer(data, dtype=np.float32)


@dataclass
class SpeakerProfile:
    """In-memory speaker profile for matching."""
    id: str
    name: str
    centroid: Optional[np.ndarray] = None
    samples: List[np.ndarray] = field(default_factory=list)
    sample_count: int = 0
    confidence: float = 0.0

    def add_sample(self, embedding: np.ndarray, quality_score: float = 1.0):
        """Add new embedding sample and update centroid."""
        # Normalize embedding
        embedding = embedding / (np.linalg.norm(embedding) + 1e-8)

        # Add to samples
        self.samples.append(embedding)
        self.sample_count += 1

        # Prune if too many samples
        if len(self.samples) > MAX_SAMPLES_PER_SPEAKER:
            self._prune_samples()

        # Update centroid with exponential moving average
        if self.centroid is None:
            self.centroid = embedding.copy()
        else:
            self.centroid = (
                ADAPTATION_RATE * self.centroid +
                (1 - ADAPTATION_RATE) * embedding
            )

        # Normalize centroid
        self.centroid = self.centroid / (np.linalg.norm(self.centroid) + 1e-8)

        # Update confidence based on sample count
        self.confidence = min(1.0, self.sample_count / 20)

    def _prune_samples(self):
        """Remove most similar samples to maintain diversity."""
        if len(self.samples) <= MAX_SAMPLES_PER_SPEAKER:
            return

        # Compute pairwise similarities
        n = len(self.samples)
        avg_sims = np.zeros(n)

        for i in range(n):
            for j in range(n):
                if i != j:
                    sim = np.dot(self.samples[i], self.samples[j])
                    avg_sims[i] += sim
            avg_sims[i] /= (n - 1)

        # Remove most redundant samples
        while len(self.samples) > MAX_SAMPLES_PER_SPEAKER:
            redundant_idx = np.argmax(avg_sims)
            self.samples.pop(redundant_idx)
            avg_sims = np.delete(avg_sims, redundant_idx)


class SpeakerEmbeddingExtractor:
    """Extract voice embeddings using ECAPA-TDNN model."""

    def __init__(self, device: Optional[str] = None):
        self._model = None
        self._device = device

    async def _load_model(self):
        """Lazy load the embedding model."""
        if self._model is not None:
            return

        def load():
            try:
                from speechbrain.inference.speaker import EncoderClassifier
                import torch

                # Determine device
                if self._device:
                    device = self._device
                elif torch.backends.mps.is_available():
                    device = "mps"
                elif torch.cuda.is_available():
                    device = "cuda"
                else:
                    device = "cpu"

                logger.info(f"Loading speaker embedding model on {device}")

                model = EncoderClassifier.from_hparams(
                    source="speechbrain/spkrec-ecapa-voxceleb",
                    savedir="models/ecapa",
                    run_opts={"device": device},
                )
                return model
            except Exception as e:
                logger.error(f"Failed to load embedding model: {e}")
                raise

        self._model = await asyncio.get_event_loop().run_in_executor(None, load)

    async def extract(
        self,
        audio: np.ndarray,
        sample_rate: int = 16000,
    ) -> np.ndarray:
        """
        Extract 192-dim embedding from audio segment.

        Args:
            audio: Audio waveform as numpy array
            sample_rate: Audio sample rate

        Returns:
            192-dimensional embedding vector
        """
        await self._load_model()

        def run_extraction():
            import torch

            # Ensure mono
            if len(audio.shape) > 1:
                audio_mono = audio.mean(axis=1)
            else:
                audio_mono = audio

            # Normalize
            audio_norm = audio_mono / (np.abs(audio_mono).max() + 1e-8)

            # Convert to tensor
            waveform = torch.tensor(audio_norm).unsqueeze(0).float()

            # Extract embedding
            embedding = self._model.encode_batch(waveform)
            return embedding.squeeze().cpu().numpy()

        embedding = await asyncio.get_event_loop().run_in_executor(None, run_extraction)
        return embedding

    def compute_similarity(self, emb1: np.ndarray, emb2: np.ndarray) -> float:
        """Compute cosine similarity between two embeddings."""
        return float(np.dot(emb1, emb2) / (np.linalg.norm(emb1) * np.linalg.norm(emb2) + 1e-8))


class SpeakerMatcher:
    """Match unknown speakers against known profiles."""

    def __init__(self, profiles: List[SpeakerProfile]):
        self.profiles = {p.id: p for p in profiles}

    def find_matches(
        self,
        embedding: np.ndarray,
    ) -> List[Tuple[str, str, float, str]]:
        """
        Find matching speakers for an unknown embedding.

        Args:
            embedding: Voice embedding to match

        Returns:
            List of (speaker_id, speaker_name, similarity, confidence_level)
            sorted by similarity descending
        """
        # Normalize input embedding
        embedding = embedding / (np.linalg.norm(embedding) + 1e-8)

        matches = []
        for profile in self.profiles.values():
            if profile.centroid is None:
                continue

            similarity = float(np.dot(embedding, profile.centroid))

            if similarity >= SUGGESTION_THRESHOLD:
                if similarity >= HIGH_CONFIDENCE_THRESHOLD:
                    confidence = "high"
                else:
                    confidence = "medium"

                matches.append((
                    profile.id,
                    profile.name,
                    similarity,
                    confidence,
                ))

        # Sort by similarity descending
        matches.sort(key=lambda x: x[2], reverse=True)
        return matches

    def identify_speaker(
        self,
        embedding: np.ndarray,
    ) -> Optional[Tuple[str, str, float]]:
        """
        Automatically identify speaker if confidence is high enough.

        Args:
            embedding: Voice embedding to identify

        Returns:
            (speaker_id, speaker_name, similarity) if confident, None otherwise
        """
        matches = self.find_matches(embedding)

        if matches and matches[0][3] == "high":
            return (matches[0][0], matches[0][1], matches[0][2])

        return None


class SpeakerLearningService:
    """Orchestrates the speaker learning workflow."""

    def __init__(self):
        self.extractor = SpeakerEmbeddingExtractor()
        self._profiles_cache: dict[str, SpeakerProfile] = {}

    async def extract_embedding(
        self,
        audio: np.ndarray,
        sample_rate: int = 16000,
    ) -> np.ndarray:
        """Extract voice embedding from audio."""
        return await self.extractor.extract(audio, sample_rate)

    async def load_profiles(self, db_session) -> List[SpeakerProfile]:
        """Load all speaker profiles from database."""
        from sqlalchemy import select
        from ..models.database import Speaker, SpeakerEmbedding

        stmt = select(Speaker)
        result = await db_session.execute(stmt)
        speakers = result.scalars().all()

        profiles = []
        for speaker in speakers:
            centroid = None
            if speaker.embedding_centroid:
                centroid = bytes_to_embedding(speaker.embedding_centroid)

            # Load sample embeddings
            stmt = select(SpeakerEmbedding).where(
                SpeakerEmbedding.speaker_id == speaker.id
            )
            result = await db_session.execute(stmt)
            embeddings = result.scalars().all()

            samples = [bytes_to_embedding(e.embedding) for e in embeddings]

            profile = SpeakerProfile(
                id=speaker.id,
                name=speaker.name,
                centroid=centroid,
                samples=samples,
                sample_count=speaker.sample_count,
                confidence=speaker.confidence,
            )
            profiles.append(profile)
            self._profiles_cache[speaker.id] = profile

        return profiles

    async def match_speaker(
        self,
        embedding: np.ndarray,
        db_session,
    ) -> Optional[Tuple[str, str, float]]:
        """
        Try to match an embedding to a known speaker.

        Returns:
            (speaker_id, speaker_name, similarity) if match found, None otherwise
        """
        profiles = await self.load_profiles(db_session)
        matcher = SpeakerMatcher(profiles)
        return matcher.identify_speaker(embedding)

    async def get_suggestions(
        self,
        embedding: np.ndarray,
        db_session,
    ) -> List[Tuple[str, str, float, str]]:
        """
        Get speaker suggestions for an unknown embedding.

        Returns:
            List of (speaker_id, speaker_name, similarity, confidence) matches
        """
        profiles = await self.load_profiles(db_session)
        matcher = SpeakerMatcher(profiles)
        return matcher.find_matches(embedding)

    async def update_speaker_profile(
        self,
        speaker_id: str,
        embedding: np.ndarray,
        db_session,
        segment_id: Optional[str] = None,
    ):
        """
        Add a new embedding sample to a speaker's profile.

        Args:
            speaker_id: The speaker to update
            embedding: New voice embedding
            db_session: Database session
            segment_id: Optional source segment ID
        """
        from sqlalchemy import select
        from ..models.database import Speaker, SpeakerEmbedding
        import uuid

        # Load or create profile
        if speaker_id in self._profiles_cache:
            profile = self._profiles_cache[speaker_id]
        else:
            stmt = select(Speaker).where(Speaker.id == speaker_id)
            result = await db_session.execute(stmt)
            speaker = result.scalar_one_or_none()

            if not speaker:
                raise ValueError(f"Speaker {speaker_id} not found")

            centroid = None
            if speaker.embedding_centroid:
                centroid = bytes_to_embedding(speaker.embedding_centroid)

            profile = SpeakerProfile(
                id=speaker.id,
                name=speaker.name,
                centroid=centroid,
                sample_count=speaker.sample_count,
                confidence=speaker.confidence,
            )
            self._profiles_cache[speaker_id] = profile

        # Add sample to profile
        profile.add_sample(embedding)

        # Store embedding in database
        new_embedding = SpeakerEmbedding(
            id=str(uuid.uuid4()),
            speaker_id=speaker_id,
            embedding=embedding_to_bytes(embedding),
            source_segment_id=segment_id,
            quality_score=1.0,
        )
        db_session.add(new_embedding)

        # Update speaker record
        stmt = select(Speaker).where(Speaker.id == speaker_id)
        result = await db_session.execute(stmt)
        speaker = result.scalar_one()

        speaker.embedding_centroid = embedding_to_bytes(profile.centroid)
        speaker.sample_count = profile.sample_count
        speaker.confidence = profile.confidence
        speaker.last_seen_at = datetime.utcnow()

        await db_session.commit()

    async def process_labeling(
        self,
        segment_ids: List[str],
        temp_speaker_id: str,
        speaker_id: Optional[str],
        new_speaker_name: Optional[str],
        db_session,
        audio_loader,  # Callable to load audio for a segment
    ) -> Tuple[str, int]:
        """
        Process user's speaker labeling action.

        Args:
            segment_ids: Segments to label
            temp_speaker_id: The temporary speaker ID being labeled
            speaker_id: Existing speaker ID (if assigning to existing)
            new_speaker_name: Name for new speaker (if creating new)
            db_session: Database session
            audio_loader: Async function to load audio for a segment

        Returns:
            (speaker_id, updated_count)
        """
        from sqlalchemy import select, update
        from ..models.database import Speaker, Segment
        import uuid

        # Create or get speaker
        if speaker_id:
            stmt = select(Speaker).where(Speaker.id == speaker_id)
            result = await db_session.execute(stmt)
            speaker = result.scalar_one_or_none()
            if not speaker:
                raise ValueError(f"Speaker {speaker_id} not found")
        else:
            if not new_speaker_name:
                raise ValueError("Either speaker_id or new_speaker_name required")

            # Create new speaker
            speaker = Speaker(
                id=str(uuid.uuid4()),
                name=new_speaker_name,
                first_seen_at=datetime.utcnow(),
            )
            db_session.add(speaker)
            await db_session.flush()

        # Get segments to update
        stmt = select(Segment).where(Segment.id.in_(segment_ids))
        result = await db_session.execute(stmt)
        segments = result.scalars().all()

        # Process each segment
        for segment in segments:
            # Load audio and extract embedding
            try:
                audio = await audio_loader(segment)
                if audio is not None:
                    embedding = await self.extract_embedding(audio)

                    # Update speaker profile
                    await self.update_speaker_profile(
                        speaker.id,
                        embedding,
                        db_session,
                        segment.id,
                    )

                    # Store embedding with segment
                    segment.embedding = embedding_to_bytes(embedding)
            except Exception as e:
                logger.warning(f"Failed to extract embedding for segment {segment.id}: {e}")

            # Update segment
            segment.speaker_id = speaker.id
            segment.is_user_verified = True
            segment.speaker_confidence = 1.0

        await db_session.commit()

        return speaker.id, len(segments)

    async def rematch_unidentified_segments(
        self,
        recording_id: str,
        db_session,
    ) -> int:
        """
        Re-run speaker matching for unidentified segments in a recording.

        Returns:
            Number of segments that were auto-matched
        """
        from sqlalchemy import select
        from ..models.database import Segment

        # Load all profiles
        profiles = await self.load_profiles(db_session)
        if not profiles:
            return 0

        matcher = SpeakerMatcher(profiles)

        # Get unidentified segments with embeddings
        stmt = select(Segment).where(
            Segment.recording_id == recording_id,
            Segment.speaker_id.is_(None),
            Segment.embedding.isnot(None),
        )
        result = await db_session.execute(stmt)
        segments = result.scalars().all()

        matched_count = 0
        for segment in segments:
            embedding = bytes_to_embedding(segment.embedding)
            match = matcher.identify_speaker(embedding)

            if match:
                speaker_id, speaker_name, similarity = match
                segment.speaker_id = speaker_id
                segment.speaker_confidence = similarity
                segment.is_user_verified = False
                matched_count += 1

        await db_session.commit()
        return matched_count
