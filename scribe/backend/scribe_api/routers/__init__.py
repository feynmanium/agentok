"""Scribe API routers."""

from .recordings import router as recordings_router
from .speakers import router as speakers_router
from .segments import router as segments_router

__all__ = [
    "recordings_router",
    "speakers_router",
    "segments_router",
]
