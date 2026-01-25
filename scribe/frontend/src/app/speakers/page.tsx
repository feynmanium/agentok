'use client';

import { useState, useCallback } from 'react';
import useSWR, { mutate } from 'swr';
import { ArrowLeft, User, Trash2, Edit2, Check, X } from 'lucide-react';
import Link from 'next/link';
import {
  getSpeakers,
  createSpeaker,
  updateSpeaker,
  deleteSpeaker,
} from '@/lib/api';
import { formatRelativeTime, cn } from '@/lib/utils';
import type { Speaker } from '@/types';

export default function SpeakersPage() {
  const { data: speakers, error, isLoading } = useSWR('/speakers', getSpeakers);
  const [newSpeakerName, setNewSpeakerName] = useState('');
  const [isCreating, setIsCreating] = useState(false);

  const handleCreateSpeaker = useCallback(async () => {
    if (!newSpeakerName.trim()) return;

    setIsCreating(true);
    try {
      await createSpeaker(newSpeakerName.trim());
      setNewSpeakerName('');
      mutate('/speakers');
    } catch (err) {
      console.error('Failed to create speaker:', err);
    } finally {
      setIsCreating(false);
    }
  }, [newSpeakerName]);

  const handleDeleteSpeaker = useCallback(async (speaker: Speaker) => {
    if (!confirm(`Delete "${speaker.name}"? Segments will become unassigned.`)) {
      return;
    }

    try {
      await deleteSpeaker(speaker.id);
      mutate('/speakers');
    } catch (err) {
      console.error('Failed to delete speaker:', err);
    }
  }, []);

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b p-4">
        <div className="max-w-4xl mx-auto flex items-center gap-4">
          <Link href="/" className="p-2 hover:bg-muted rounded-lg">
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <div>
            <h1 className="font-semibold text-lg">Manage Speakers</h1>
            <p className="text-sm text-muted-foreground">
              View and manage speaker profiles for automatic identification
            </p>
          </div>
        </div>
      </header>

      <main className="max-w-4xl mx-auto p-6">
        {/* Create New Speaker */}
        <div className="mb-8">
          <h2 className="font-medium mb-3">Add New Speaker</h2>
          <div className="flex gap-2">
            <input
              type="text"
              value={newSpeakerName}
              onChange={(e) => setNewSpeakerName(e.target.value)}
              placeholder="Enter speaker name..."
              className="flex-1 px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20"
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreateSpeaker();
              }}
            />
            <button
              onClick={handleCreateSpeaker}
              disabled={isCreating || !newSpeakerName.trim()}
              className="px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 disabled:opacity-50"
            >
              {isCreating ? 'Adding...' : 'Add Speaker'}
            </button>
          </div>
        </div>

        {/* Speaker List */}
        <div>
          <h2 className="font-medium mb-3">
            Known Speakers ({speakers?.length || 0})
          </h2>

          {isLoading && (
            <div className="text-center text-muted-foreground py-8">
              Loading speakers...
            </div>
          )}

          {error && (
            <div className="text-center text-red-500 py-8">
              Failed to load speakers
            </div>
          )}

          {speakers && speakers.length === 0 && (
            <div className="text-center text-muted-foreground py-8 border rounded-lg bg-muted/30">
              <User className="w-8 h-8 mx-auto mb-2 opacity-50" />
              <p>No speakers yet</p>
              <p className="text-sm mt-1">
                Speakers are created when you label them in recordings
              </p>
            </div>
          )}

          {speakers && speakers.length > 0 && (
            <div className="space-y-2">
              {speakers.map((speaker) => (
                <SpeakerCard
                  key={speaker.id}
                  speaker={speaker}
                  onDelete={() => handleDeleteSpeaker(speaker)}
                />
              ))}
            </div>
          )}
        </div>
      </main>
    </div>
  );
}

function SpeakerCard({
  speaker,
  onDelete,
}: {
  speaker: Speaker;
  onDelete: () => void;
}) {
  const [isEditing, setIsEditing] = useState(false);
  const [editName, setEditName] = useState(speaker.name);
  const [isSaving, setIsSaving] = useState(false);

  const handleSave = async () => {
    if (!editName.trim() || editName === speaker.name) {
      setIsEditing(false);
      setEditName(speaker.name);
      return;
    }

    setIsSaving(true);
    try {
      await updateSpeaker(speaker.id, { name: editName.trim() });
      mutate('/speakers');
      setIsEditing(false);
    } catch (err) {
      console.error('Failed to update speaker:', err);
      setEditName(speaker.name);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="flex items-center gap-4 p-4 border rounded-lg hover:border-primary/30 transition-colors group">
      {/* Avatar */}
      <div
        className="w-12 h-12 rounded-full flex items-center justify-center text-white text-lg font-medium flex-shrink-0"
        style={{ backgroundColor: speaker.color || '#6B7280' }}
      >
        {speaker.name.charAt(0).toUpperCase()}
      </div>

      {/* Info */}
      <div className="flex-1 min-w-0">
        {isEditing ? (
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
              className="px-2 py-1 border rounded focus:outline-none focus:ring-2 focus:ring-primary/20"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSave();
                if (e.key === 'Escape') {
                  setIsEditing(false);
                  setEditName(speaker.name);
                }
              }}
            />
            <button
              onClick={handleSave}
              disabled={isSaving}
              className="p-1 hover:bg-green-100 text-green-600 rounded"
            >
              <Check className="w-4 h-4" />
            </button>
            <button
              onClick={() => {
                setIsEditing(false);
                setEditName(speaker.name);
              }}
              className="p-1 hover:bg-muted rounded"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        ) : (
          <>
            <p className="font-medium">{speaker.name}</p>
            <div className="flex items-center gap-4 text-sm text-muted-foreground">
              <span>{speaker.sample_count} voice samples</span>
              {speaker.confidence > 0 && (
                <span>{Math.round(speaker.confidence * 100)}% confidence</span>
              )}
              {speaker.last_seen_at && (
                <span>Last seen {formatRelativeTime(speaker.last_seen_at)}</span>
              )}
            </div>
          </>
        )}
      </div>

      {/* Actions */}
      {!isEditing && (
        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            onClick={() => setIsEditing(true)}
            className="p-2 hover:bg-muted rounded-lg"
            title="Edit name"
          >
            <Edit2 className="w-4 h-4" />
          </button>
          <button
            onClick={onDelete}
            className="p-2 hover:bg-red-100 hover:text-red-600 rounded-lg"
            title="Delete speaker"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      )}
    </div>
  );
}
