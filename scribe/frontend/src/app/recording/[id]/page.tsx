'use client';

import { useState, useCallback, useMemo } from 'react';
import { useParams, useRouter } from 'next/navigation';
import useSWR, { mutate } from 'swr';
import {
  ArrowLeft,
  Download,
  Search,
  CheckCircle,
  AlertCircle,
  User,
  Clock,
} from 'lucide-react';
import Link from 'next/link';
import {
  getRecording,
  getSpeakers,
  labelSegments,
  exportRecording,
} from '@/lib/api';
import {
  formatDuration,
  formatTimestamp,
  downloadBlob,
  cn,
} from '@/lib/utils';
import type { Segment, Speaker, SpeakerSummary, ExportFormat } from '@/types';
import { SpeakerLabelModal } from '@/components/SpeakerLabelModal';

export default function RecordingPage() {
  const params = useParams();
  const router = useRouter();
  const recordingId = params.id as string;

  const { data: recording, error, isLoading } = useSWR(
    `/recordings/${recordingId}`,
    () => getRecording(recordingId)
  );
  const { data: speakers } = useSWR('/speakers', getSpeakers);

  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTempSpeaker, setSelectedTempSpeaker] = useState<string | null>(null);
  const [isExporting, setIsExporting] = useState(false);

  // Filter segments by search
  const filteredSegments = useMemo(() => {
    if (!recording?.segments) return [];
    if (!searchQuery) return recording.segments;

    const query = searchQuery.toLowerCase();
    return recording.segments.filter(
      (seg) =>
        seg.text.toLowerCase().includes(query) ||
        seg.speaker?.name.toLowerCase().includes(query) ||
        seg.temp_speaker_id?.toLowerCase().includes(query)
    );
  }, [recording?.segments, searchQuery]);

  // Get unique temp speaker IDs that need labeling
  const unlabeledSpeakers = useMemo(() => {
    if (!recording?.segments) return [];

    const tempIds = new Set<string>();
    for (const seg of recording.segments) {
      if (!seg.speaker_id && seg.temp_speaker_id) {
        tempIds.add(seg.temp_speaker_id);
      }
    }
    return Array.from(tempIds).sort();
  }, [recording?.segments]);

  const handleExport = useCallback(async (format: ExportFormat) => {
    if (!recording) return;

    setIsExporting(true);
    try {
      const blob = await exportRecording(recording.id, format);
      const extension = format;
      downloadBlob(blob, `${recording.title}.${extension}`);
    } catch (err) {
      console.error('Export failed:', err);
    } finally {
      setIsExporting(false);
    }
  }, [recording]);

  const handleLabelComplete = useCallback(() => {
    setSelectedTempSpeaker(null);
    mutate(`/recordings/${recordingId}`);
    mutate('/speakers');
  }, [recordingId]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-muted-foreground">Loading transcript...</div>
      </div>
    );
  }

  if (error || !recording) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-center">
          <p className="text-red-500 mb-4">Failed to load recording</p>
          <Link href="/" className="text-primary hover:underline">
            Back to recordings
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen">
      {/* Main Transcript View */}
      <main className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <header className="border-b p-4 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link href="/" className="p-2 hover:bg-muted rounded-lg">
              <ArrowLeft className="w-5 h-5" />
            </Link>
            <div>
              <h1 className="font-semibold">{recording.title}</h1>
              <div className="flex items-center gap-3 text-sm text-muted-foreground">
                {recording.duration_ms && (
                  <span className="flex items-center gap-1">
                    <Clock className="w-3 h-3" />
                    {formatDuration(recording.duration_ms)}
                  </span>
                )}
                <span>
                  {recording.speaker_summary?.length || 0} speakers
                </span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {/* Export Dropdown */}
            <div className="relative group">
              <button
                className="px-3 py-2 text-sm border rounded-lg hover:bg-muted flex items-center gap-2"
                disabled={isExporting}
              >
                <Download className="w-4 h-4" />
                {isExporting ? 'Exporting...' : 'Export'}
              </button>
              <div className="absolute right-0 mt-1 bg-white border rounded-lg shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all z-10">
                {(['txt', 'srt', 'vtt', 'json'] as ExportFormat[]).map((format) => (
                  <button
                    key={format}
                    onClick={() => handleExport(format)}
                    className="block w-full px-4 py-2 text-left text-sm hover:bg-muted first:rounded-t-lg last:rounded-b-lg"
                  >
                    Export as .{format.toUpperCase()}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </header>

        {/* Search Bar */}
        <div className="p-4 border-b">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search transcript..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20"
            />
          </div>
        </div>

        {/* Transcript Segments */}
        <div className="flex-1 overflow-auto p-4">
          {filteredSegments.length === 0 ? (
            <div className="text-center text-muted-foreground py-12">
              {searchQuery ? 'No matching segments found' : 'No segments yet'}
            </div>
          ) : (
            <div className="space-y-4">
              {filteredSegments.map((segment) => (
                <SegmentCard
                  key={segment.id}
                  segment={segment}
                  onLabelClick={
                    !segment.speaker_id && segment.temp_speaker_id
                      ? () => setSelectedTempSpeaker(segment.temp_speaker_id!)
                      : undefined
                  }
                />
              ))}
            </div>
          )}
        </div>
      </main>

      {/* Sidebar - Speaker Summary */}
      <aside className="w-80 border-l bg-muted/30 p-4 overflow-auto">
        <h2 className="font-semibold mb-4">Speakers</h2>

        {/* Unlabeled Speakers Alert */}
        {unlabeledSpeakers.length > 0 && (
          <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
            <div className="flex items-start gap-2">
              <AlertCircle className="w-4 h-4 text-yellow-600 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-yellow-800">
                  {unlabeledSpeakers.length} speaker{unlabeledSpeakers.length !== 1 ? 's' : ''} need labeling
                </p>
                <p className="text-xs text-yellow-700 mt-1">
                  Click on a speaker below to assign an identity
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Speaker List */}
        <div className="space-y-2">
          {recording.speaker_summary?.map((summary) => (
            <SpeakerSummaryCard
              key={summary.speaker_id || summary.temp_speaker_id}
              summary={summary}
              isLabeled={!!summary.speaker_id}
              onLabelClick={
                !summary.speaker_id && summary.temp_speaker_id
                  ? () => setSelectedTempSpeaker(summary.temp_speaker_id!)
                  : undefined
              }
            />
          ))}
        </div>
      </aside>

      {/* Speaker Label Modal */}
      {selectedTempSpeaker && (
        <SpeakerLabelModal
          recordingId={recordingId}
          tempSpeakerId={selectedTempSpeaker}
          segments={recording.segments.filter(
            (s) => s.temp_speaker_id === selectedTempSpeaker && !s.speaker_id
          )}
          existingSpeakers={speakers || []}
          onClose={() => setSelectedTempSpeaker(null)}
          onComplete={handleLabelComplete}
        />
      )}
    </div>
  );
}

function SegmentCard({
  segment,
  onLabelClick,
}: {
  segment: Segment;
  onLabelClick?: () => void;
}) {
  const speakerName = segment.speaker?.name || segment.temp_speaker_id || 'Unknown';
  const speakerColor = segment.speaker?.color || '#6B7280';
  const isLabeled = !!segment.speaker_id;

  return (
    <div className="flex gap-3 group">
      {/* Timestamp */}
      <div className="text-xs text-muted-foreground w-14 pt-1 flex-shrink-0">
        {formatTimestamp(segment.start_ms)}
      </div>

      {/* Content */}
      <div className="flex-1">
        {/* Speaker Header */}
        <div className="flex items-center gap-2 mb-1">
          <span
            className="inline-flex items-center gap-1.5 text-sm font-medium"
            style={{ color: speakerColor }}
          >
            <span
              className="w-2 h-2 rounded-full"
              style={{ backgroundColor: speakerColor }}
            />
            {speakerName}
          </span>

          {isLabeled ? (
            <CheckCircle className="w-3 h-3 text-green-500" />
          ) : onLabelClick ? (
            <button
              onClick={onLabelClick}
              className="text-xs text-primary hover:underline opacity-0 group-hover:opacity-100 transition-opacity"
            >
              Label speaker
            </button>
          ) : null}
        </div>

        {/* Text */}
        <p className="text-sm leading-relaxed">{segment.text}</p>
      </div>
    </div>
  );
}

function SpeakerSummaryCard({
  summary,
  isLabeled,
  onLabelClick,
}: {
  summary: SpeakerSummary;
  isLabeled: boolean;
  onLabelClick?: () => void;
}) {
  const color = summary.color || '#6B7280';

  return (
    <div
      className={cn(
        'p-3 rounded-lg border transition-colors',
        onLabelClick ? 'cursor-pointer hover:border-primary' : '',
        !isLabeled ? 'border-yellow-200 bg-yellow-50/50' : ''
      )}
      onClick={onLabelClick}
    >
      <div className="flex items-center gap-2 mb-2">
        <span
          className="w-3 h-3 rounded-full"
          style={{ backgroundColor: color }}
        />
        <span className="font-medium text-sm">
          {summary.speaker_name}
        </span>
        {isLabeled ? (
          <CheckCircle className="w-3 h-3 text-green-500 ml-auto" />
        ) : (
          <AlertCircle className="w-3 h-3 text-yellow-500 ml-auto" />
        )}
      </div>

      <div className="grid grid-cols-2 gap-2 text-xs text-muted-foreground">
        <div>
          <span className="font-medium text-foreground">
            {summary.percentage.toFixed(0)}%
          </span>{' '}
          of conversation
        </div>
        <div>
          <span className="font-medium text-foreground">
            {summary.segment_count}
          </span>{' '}
          segments
        </div>
        <div>
          <span className="font-medium text-foreground">
            {formatDuration(summary.total_duration_ms)}
          </span>{' '}
          speaking
        </div>
        <div>
          <span className="font-medium text-foreground">
            {summary.word_count}
          </span>{' '}
          words
        </div>
      </div>
    </div>
  );
}
