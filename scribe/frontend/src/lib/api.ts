import type {
  Recording,
  RecordingWithSegments,
  Speaker,
  Segment,
  SpeakerLabelRequest,
  SpeakerMatch,
  ExportFormat,
} from '@/types';

const API_BASE = '/api/v1';

async function fetchAPI<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const res = await fetch(`${API_BASE}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
    ...options,
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({ detail: 'Unknown error' }));
    throw new Error(error.detail || 'API request failed');
  }

  return res.json();
}

// Recordings API
export async function getRecordings(): Promise<Recording[]> {
  return fetchAPI<Recording[]>('/recordings');
}

export async function getRecording(id: string): Promise<RecordingWithSegments> {
  return fetchAPI<RecordingWithSegments>(`/recordings/${id}`);
}

export async function uploadRecording(
  file: File,
  title: string,
  options?: {
    description?: string;
    enableDiarization?: boolean;
    language?: string;
  }
): Promise<Recording> {
  const formData = new FormData();
  formData.append('audio_file', file);
  formData.append('title', title);
  if (options?.description) {
    formData.append('description', options.description);
  }
  formData.append('enable_diarization', String(options?.enableDiarization ?? true));
  formData.append('language', options?.language ?? 'auto');

  const res = await fetch(`${API_BASE}/recordings`, {
    method: 'POST',
    body: formData,
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({ detail: 'Upload failed' }));
    throw new Error(error.detail);
  }

  return res.json();
}

export async function deleteRecording(id: string): Promise<void> {
  await fetchAPI(`/recordings/${id}`, { method: 'DELETE' });
}

export async function exportRecording(
  id: string,
  format: ExportFormat,
  options?: {
    includeTimestamps?: boolean;
    includeSpeakerNames?: boolean;
  }
): Promise<Blob> {
  const res = await fetch(`${API_BASE}/recordings/${id}/export`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      format,
      include_timestamps: options?.includeTimestamps ?? true,
      include_speaker_names: options?.includeSpeakerNames ?? true,
    }),
  });

  if (!res.ok) {
    throw new Error('Export failed');
  }

  return res.blob();
}

// Speakers API
export async function getSpeakers(): Promise<Speaker[]> {
  return fetchAPI<Speaker[]>('/speakers');
}

export async function createSpeaker(name: string, email?: string): Promise<Speaker> {
  return fetchAPI<Speaker>('/speakers', {
    method: 'POST',
    body: JSON.stringify({ name, email }),
  });
}

export async function updateSpeaker(
  id: string,
  data: { name?: string; email?: string; color?: string }
): Promise<Speaker> {
  return fetchAPI<Speaker>(`/speakers/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

export async function deleteSpeaker(id: string): Promise<void> {
  await fetchAPI(`/speakers/${id}`, { method: 'DELETE' });
}

export async function mergeSpeakers(
  targetId: string,
  sourceId: string
): Promise<{ status: string; merged_segments: number }> {
  return fetchAPI(`/speakers/${targetId}/merge?source_speaker_id=${sourceId}`, {
    method: 'POST',
  });
}

// Segments API
export async function updateSegment(
  id: string,
  data: { text?: string; speaker_id?: string; is_user_verified?: boolean }
): Promise<Segment> {
  return fetchAPI<Segment>(`/segments/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

export async function labelSegments(request: SpeakerLabelRequest): Promise<{
  updated_count: number;
  speaker: Speaker;
}> {
  return fetchAPI('/segments/bulk-label', {
    method: 'POST',
    body: JSON.stringify(request),
  });
}

export async function getSpeakerSuggestions(
  segmentId: string
): Promise<SpeakerMatch[]> {
  return fetchAPI<SpeakerMatch[]>(`/segments/${segmentId}/suggestions`);
}

export async function searchSegments(
  query: string,
  options?: {
    recordingId?: string;
    speakerId?: string;
    limit?: number;
  }
): Promise<{
  results: Array<{
    segment: Segment;
    recording: { id: string; title: string };
    highlight: string;
  }>;
  total: number;
}> {
  const params = new URLSearchParams({ q: query });
  if (options?.recordingId) params.append('recording_id', options.recordingId);
  if (options?.speakerId) params.append('speaker_id', options.speakerId);
  if (options?.limit) params.append('limit', String(options.limit));

  return fetchAPI(`/segments/search?${params}`);
}
