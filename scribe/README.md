# Scribe - Mac Screen Transcription

An Otter.ai-like transcription application for macOS that transcribes audio from recordings, performs speaker diarization, and learns speaker identities from your manual labels for automatic recognition in future sessions.

## Features

- **Audio File Transcription** - Upload MP3, WAV, M4A, MP4, WebM files
- **Speaker Diarization** - Automatically segments audio by speaker
- **Speaker Learning** - Label "Speaker A" as "Bill" and the system learns Bill's voice
- **Automatic Recognition** - Known speakers are automatically identified in new recordings
- **Export Options** - Export transcripts as TXT, SRT, VTT, or JSON
- **Searchable Transcripts** - Full-text search across all recordings

## Architecture

```
scribe/
├── backend/          # FastAPI Python backend
│   └── scribe_api/
│       ├── models/       # SQLite database models
│       ├── routers/      # API endpoints
│       └── services/     # Transcription, diarization, speaker learning
└── frontend/         # Next.js React frontend
    └── src/
        ├── app/          # Pages (recordings, transcript view, speakers)
        ├── components/   # React components
        └── lib/          # API client, utilities
```

## Tech Stack

### Backend
- **FastAPI** - Modern async Python web framework
- **SQLite** - Local database with async support
- **OpenAI Whisper** - Speech-to-text transcription
- **pyannote.audio** - Speaker diarization
- **SpeechBrain ECAPA-TDNN** - Voice embeddings for speaker identification

### Frontend
- **Next.js 15** - React framework
- **TailwindCSS** - Styling
- **Radix UI** - Accessible components
- **SWR** - Data fetching

## Quick Start

### Prerequisites

- Python 3.11+
- Node.js 18+
- OpenAI API key (for Whisper transcription)
- Hugging Face token (for pyannote diarization)

### Backend Setup

```bash
cd scribe/backend

# Create virtual environment
python -m venv venv
source venv/bin/activate  # or `venv\Scripts\activate` on Windows

# Install dependencies
pip install -e .

# Copy environment file
cp .env.example .env

# Edit .env and add your API keys:
# - OPENAI_API_KEY=sk-your-key
# - HF_TOKEN=hf_your-token

# Run the server
uvicorn scribe_api.main:app --reload --port 5005
```

### Frontend Setup

```bash
cd scribe/frontend

# Install dependencies
npm install
# or
pnpm install

# Run development server
npm run dev
```

### Access the App

- Frontend: http://localhost:3001
- API Docs: http://localhost:5005/docs

## Usage

### 1. Upload a Recording

Click "Drop audio file or click to upload" and select an audio or video file. Supported formats: MP3, WAV, M4A, MP4, WebM, OGG, FLAC.

The file will be processed in the background:
1. Audio extraction (if video)
2. Speaker diarization (who spoke when)
3. Transcription with Whisper
4. Speaker matching against known profiles

### 2. Label Speakers

In the transcript view, you'll see speakers labeled as "Speaker A", "Speaker B", etc.

Click "Label speaker" or click on a speaker in the sidebar to assign a real identity:
- Select an existing speaker from your library
- Or create a new speaker profile

When you label a speaker, the system:
1. Updates all their segments in the recording
2. Extracts voice embeddings from their audio
3. Adds embeddings to the speaker's profile
4. Uses this for automatic identification in future recordings

### 3. Export Transcripts

Click the Export button to download the transcript in various formats:
- **TXT** - Plain text with timestamps and speaker names
- **SRT** - Subtitle format for video editing
- **VTT** - Web Video Text Tracks
- **JSON** - Full data with word-level timestamps

## API Reference

### Recordings

- `GET /api/v1/recordings` - List all recordings
- `POST /api/v1/recordings` - Upload new recording
- `GET /api/v1/recordings/{id}` - Get recording with segments
- `DELETE /api/v1/recordings/{id}` - Delete recording
- `POST /api/v1/recordings/{id}/export` - Export transcript

### Speakers

- `GET /api/v1/speakers` - List all speakers
- `POST /api/v1/speakers` - Create speaker
- `PATCH /api/v1/speakers/{id}` - Update speaker
- `DELETE /api/v1/speakers/{id}` - Delete speaker
- `POST /api/v1/speakers/{id}/merge` - Merge two speakers

### Segments

- `PATCH /api/v1/segments/{id}` - Update segment text or speaker
- `POST /api/v1/segments/bulk-label` - Label multiple segments
- `POST /api/v1/segments/search` - Search transcripts

## Speaker Learning Algorithm

The speaker learning system uses voice embeddings to identify speakers:

1. **Embedding Extraction**: ECAPA-TDNN extracts 192-dimensional voice embeddings from audio segments

2. **Profile Building**: When you label a speaker, embeddings are:
   - Added to the speaker's sample pool (up to 50 samples)
   - Used to compute a centroid (average embedding)
   - Stored for future matching

3. **Speaker Matching**: For new recordings:
   - Extract embedding from each speaker segment
   - Compute cosine similarity against all known speaker centroids
   - If similarity > 0.75: auto-assign (high confidence)
   - If similarity 0.60-0.75: suggest match (medium confidence)
   - Otherwise: assign temporary ID

4. **Continuous Learning**: The centroid is updated with exponential moving average to adapt to new samples while preventing drift.

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `OPENAI_API_KEY` | OpenAI API key for Whisper | Yes |
| `HF_TOKEN` | Hugging Face token for pyannote | Yes (for diarization) |
| `DATABASE_URL` | SQLite database URL | No (defaults to `./data/scribe.db`) |
| `PORT` | API server port | No (defaults to 5005) |
| `DEBUG` | Enable debug mode | No |

## Development

### Running Tests

```bash
cd scribe/backend
pytest
```

### Database

The SQLite database is created automatically at `scribe/backend/data/scribe.db`. To reset:

```bash
rm scribe/backend/data/scribe.db
# Restart the server to recreate
```

## Roadmap

- [ ] Real-time transcription during screen recording
- [ ] Mac screen capture integration (ScreenCaptureKit)
- [ ] Local Whisper inference (faster-whisper)
- [ ] Collaborative transcript viewing
- [ ] AI-powered meeting summaries
- [ ] Cloud sync (optional)

## License

MIT
