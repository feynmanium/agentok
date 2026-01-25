'use client';

import { useState, useCallback } from 'react';
import { X, User, Plus, CheckCircle } from 'lucide-react';
import { labelSegments } from '@/lib/api';
import { cn } from '@/lib/utils';
import type { Segment, Speaker } from '@/types';

interface SpeakerLabelModalProps {
  recordingId: string;
  tempSpeakerId: string;
  segments: Segment[];
  existingSpeakers: Speaker[];
  onClose: () => void;
  onComplete: () => void;
}

export function SpeakerLabelModal({
  recordingId,
  tempSpeakerId,
  segments,
  existingSpeakers,
  onClose,
  onComplete,
}: SpeakerLabelModalProps) {
  const [selectedSpeakerId, setSelectedSpeakerId] = useState<string | null>(null);
  const [newSpeakerName, setNewSpeakerName] = useState('');
  const [isCreatingNew, setIsCreatingNew] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = useCallback(async () => {
    if (!selectedSpeakerId && !newSpeakerName.trim()) {
      setError('Please select a speaker or enter a new name');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      await labelSegments({
        segment_ids: segments.map((s) => s.id),
        temp_speaker_id: tempSpeakerId,
        speaker_id: isCreatingNew ? undefined : selectedSpeakerId || undefined,
        new_speaker_name: isCreatingNew ? newSpeakerName.trim() : undefined,
        apply_to_recording: true,
      });
      onComplete();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to label speaker');
    } finally {
      setIsSubmitting(false);
    }
  }, [segments, tempSpeakerId, selectedSpeakerId, newSpeakerName, isCreatingNew, onComplete]);

  // Get sample text from segments
  const sampleText = segments[0]?.text.slice(0, 100) + (segments[0]?.text.length > 100 ? '...' : '');

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-md mx-4">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="font-semibold">Label Speaker</h2>
          <button
            onClick={onClose}
            className="p-1 hover:bg-muted rounded-lg"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-4 space-y-4">
          {/* Current Speaker Info */}
          <div>
            <p className="text-sm text-muted-foreground mb-2">
              Who is "{tempSpeakerId}"?
            </p>
            <div className="p-3 bg-muted/50 rounded-lg text-sm">
              <p className="italic">"{sampleText}"</p>
              <p className="text-xs text-muted-foreground mt-2">
                {segments.length} segment{segments.length !== 1 ? 's' : ''} will be labeled
              </p>
            </div>
          </div>

          {/* Existing Speakers */}
          {existingSpeakers.length > 0 && !isCreatingNew && (
            <div>
              <p className="text-sm font-medium mb-2">Select existing speaker:</p>
              <div className="space-y-1 max-h-40 overflow-auto">
                {existingSpeakers.map((speaker) => (
                  <button
                    key={speaker.id}
                    onClick={() => setSelectedSpeakerId(speaker.id)}
                    className={cn(
                      'w-full flex items-center gap-3 p-2 rounded-lg text-left transition-colors',
                      selectedSpeakerId === speaker.id
                        ? 'bg-primary/10 border-primary border'
                        : 'hover:bg-muted border border-transparent'
                    )}
                  >
                    <span
                      className="w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-medium"
                      style={{ backgroundColor: speaker.color || '#6B7280' }}
                    >
                      {speaker.name.charAt(0).toUpperCase()}
                    </span>
                    <div className="flex-1 min-w-0">
                      <p className="font-medium text-sm truncate">{speaker.name}</p>
                      <p className="text-xs text-muted-foreground">
                        {speaker.sample_count} samples
                        {speaker.confidence > 0 && ` - ${Math.round(speaker.confidence * 100)}% confident`}
                      </p>
                    </div>
                    {selectedSpeakerId === speaker.id && (
                      <CheckCircle className="w-5 h-5 text-primary" />
                    )}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Divider */}
          {existingSpeakers.length > 0 && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <div className="flex-1 border-t" />
              <span>OR</span>
              <div className="flex-1 border-t" />
            </div>
          )}

          {/* Create New Speaker */}
          <div>
            {!isCreatingNew ? (
              <button
                onClick={() => {
                  setIsCreatingNew(true);
                  setSelectedSpeakerId(null);
                }}
                className="w-full flex items-center gap-2 p-3 border border-dashed rounded-lg hover:border-primary hover:bg-primary/5 transition-colors text-sm"
              >
                <Plus className="w-4 h-4" />
                Create new speaker
              </button>
            ) : (
              <div>
                <label className="text-sm font-medium mb-1 block">
                  New speaker name:
                </label>
                <input
                  type="text"
                  value={newSpeakerName}
                  onChange={(e) => setNewSpeakerName(e.target.value)}
                  placeholder="Enter name (e.g., Bill, Gill)"
                  className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20"
                  autoFocus
                />
                <button
                  onClick={() => {
                    setIsCreatingNew(false);
                    setNewSpeakerName('');
                  }}
                  className="text-xs text-muted-foreground hover:text-foreground mt-2"
                >
                  Cancel - select existing speaker instead
                </button>
              </div>
            )}
          </div>

          {/* Error Message */}
          {error && (
            <p className="text-sm text-red-500">{error}</p>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-2 p-4 border-t bg-muted/30">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm hover:bg-muted rounded-lg"
            disabled={isSubmitting}
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={isSubmitting || (!selectedSpeakerId && !newSpeakerName.trim())}
            className="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSubmitting ? 'Labeling...' : 'Apply Label'}
          </button>
        </div>
      </div>
    </div>
  );
}
