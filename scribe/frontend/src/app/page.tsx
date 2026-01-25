'use client';

import { useState, useCallback } from 'react';
import useSWR, { mutate } from 'swr';
import { Upload, Mic, FileAudio, Trash2, ExternalLink } from 'lucide-react';
import Link from 'next/link';
import { getRecordings, uploadRecording, deleteRecording } from '@/lib/api';
import { formatDuration, formatRelativeTime } from '@/lib/utils';
import type { Recording } from '@/types';

export default function HomePage() {
  const { data: recordings, error, isLoading } = useSWR('/recordings', getRecordings);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState<string | null>(null);

  const handleFileUpload = useCallback(async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setIsUploading(true);
    setUploadProgress('Uploading...');

    try {
      const title = file.name.replace(/\.[^/.]+$/, ''); // Remove extension
      await uploadRecording(file, title);
      setUploadProgress('Processing...');
      mutate('/recordings');
    } catch (err) {
      console.error('Upload failed:', err);
      setUploadProgress('Upload failed');
    } finally {
      setIsUploading(false);
      setTimeout(() => setUploadProgress(null), 2000);
    }
  }, []);

  const handleDelete = useCallback(async (id: string, title: string) => {
    if (!confirm(`Delete "${title}"? This cannot be undone.`)) return;

    try {
      await deleteRecording(id);
      mutate('/recordings');
    } catch (err) {
      console.error('Delete failed:', err);
    }
  }, []);

  return (
    <div className="flex h-screen">
      {/* Sidebar */}
      <aside className="w-80 border-r bg-muted/30 p-6 flex flex-col">
        <div className="mb-8">
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Mic className="w-6 h-6" />
            Scribe
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Mac Screen Transcription
          </p>
        </div>

        {/* Upload Section */}
        <div className="space-y-4 mb-8">
          <label className="block">
            <div className="border-2 border-dashed rounded-lg p-6 text-center cursor-pointer hover:border-primary hover:bg-primary/5 transition-colors">
              <Upload className="w-8 h-8 mx-auto mb-2 text-muted-foreground" />
              <p className="text-sm font-medium">
                {isUploading ? uploadProgress : 'Drop audio file or click to upload'}
              </p>
              <p className="text-xs text-muted-foreground mt-1">
                MP3, WAV, M4A, MP4, WebM
              </p>
            </div>
            <input
              type="file"
              accept="audio/*,video/*,.mp3,.wav,.m4a,.mp4,.webm,.ogg,.flac"
              className="hidden"
              onChange={handleFileUpload}
              disabled={isUploading}
            />
          </label>
        </div>

        {/* Quick Stats */}
        <div className="text-sm text-muted-foreground">
          {recordings && (
            <p>{recordings.length} recording{recordings.length !== 1 ? 's' : ''}</p>
          )}
        </div>

        <div className="mt-auto pt-4 border-t">
          <Link
            href="/speakers"
            className="text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            Manage Speakers
          </Link>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-auto p-6">
        <h2 className="text-xl font-semibold mb-6">Recent Recordings</h2>

        {isLoading && (
          <div className="text-center text-muted-foreground py-12">
            Loading recordings...
          </div>
        )}

        {error && (
          <div className="text-center text-red-500 py-12">
            Failed to load recordings. Is the API running?
          </div>
        )}

        {recordings && recordings.length === 0 && (
          <div className="text-center text-muted-foreground py-12">
            <FileAudio className="w-12 h-12 mx-auto mb-4 opacity-50" />
            <p>No recordings yet</p>
            <p className="text-sm mt-1">Upload an audio file to get started</p>
          </div>
        )}

        {recordings && recordings.length > 0 && (
          <div className="space-y-3">
            {recordings.map((recording) => (
              <RecordingCard
                key={recording.id}
                recording={recording}
                onDelete={() => handleDelete(recording.id, recording.title)}
              />
            ))}
          </div>
        )}
      </main>
    </div>
  );
}

function RecordingCard({
  recording,
  onDelete,
}: {
  recording: Recording;
  onDelete: () => void;
}) {
  const statusColors = {
    pending: 'bg-yellow-100 text-yellow-800',
    processing: 'bg-blue-100 text-blue-800',
    completed: 'bg-green-100 text-green-800',
    failed: 'bg-red-100 text-red-800',
  };

  return (
    <div className="border rounded-lg p-4 hover:border-primary/50 transition-colors group">
      <div className="flex items-start justify-between">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3">
            <Link
              href={`/recording/${recording.id}`}
              className="font-medium hover:text-primary truncate"
            >
              {recording.title}
            </Link>
            <span
              className={`text-xs px-2 py-0.5 rounded-full ${statusColors[recording.status]}`}
            >
              {recording.status}
            </span>
          </div>

          <div className="flex items-center gap-4 mt-2 text-sm text-muted-foreground">
            {recording.duration_ms && (
              <span>{formatDuration(recording.duration_ms)}</span>
            )}
            <span>{formatRelativeTime(recording.created_at)}</span>
          </div>
        </div>

        <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
          <Link
            href={`/recording/${recording.id}`}
            className="p-2 hover:bg-muted rounded-lg"
            title="View transcript"
          >
            <ExternalLink className="w-4 h-4" />
          </Link>
          <button
            onClick={onDelete}
            className="p-2 hover:bg-red-100 hover:text-red-600 rounded-lg"
            title="Delete recording"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  );
}
