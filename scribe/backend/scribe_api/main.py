"""Scribe API - Mac screen transcription with speaker learning."""

import logging
import os
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
from fastapi.responses import JSONResponse

from .models.database import init_db
from .routers import recordings_router, speakers_router, segments_router

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan handler."""
    # Startup
    logger.info("Initializing Scribe API...")

    # Initialize database
    await init_db()
    logger.info("Database initialized")

    # Create upload directory
    os.makedirs("data/uploads", exist_ok=True)
    os.makedirs("models", exist_ok=True)

    yield

    # Shutdown
    logger.info("Shutting down Scribe API...")


app = FastAPI(
    title="Scribe API",
    description="Mac screen transcription with speaker learning",
    version="1.0.0",
    lifespan=lifespan,
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Configure appropriately for production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(recordings_router, prefix="/api/v1")
app.include_router(speakers_router, prefix="/api/v1")
app.include_router(segments_router, prefix="/api/v1")


@app.get("/")
async def root():
    """Root endpoint."""
    return {
        "name": "Scribe API",
        "version": "1.0.0",
        "docs": "/docs",
    }


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {"status": "healthy"}


@app.exception_handler(Exception)
async def global_exception_handler(request, exc):
    """Global exception handler."""
    logger.error(f"Unhandled exception: {exc}", exc_info=True)
    return JSONResponse(
        status_code=500,
        content={"detail": "Internal server error"},
    )


# Run with: uvicorn scribe_api.main:app --reload --port 5005
if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "scribe_api.main:app",
        host="0.0.0.0",
        port=int(os.getenv("PORT", "5005")),
        reload=os.getenv("DEBUG", "false").lower() == "true",
    )
