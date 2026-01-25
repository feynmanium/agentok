// Recording types
export interface Recording {
  id: string;
  title: string;
  description?: string;
  source_type: 'screen_capture' | 'audio_file' | 'live_mic';
  source_path?: string;
  duration_ms?: number;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  audio_format?: string;
  created_at: string;
  updated_at: string;
}

export interface RecordingWithSegments extends Recording {
  segments: Segment[];
  speaker_summary: SpeakerSummary[];
}

// Speaker types
export interface Speaker {
  id: string;
  name: string;
  email?: string;
  color?: string;
  sample_count: number;
  confidence: number;
  first_seen_at?: string;
  last_seen_at?: string;
  created_at: string;
}

export interface SpeakerSummary {
  speaker_id?: string;
  speaker_name: string;
  temp_speaker_id?: string;
  color?: string;
  total_duration_ms: number;
  segment_count: number;
  word_count: number;
  percentage: number;
}

export interface SpeakerMatch {
  speaker_id: string;
  speaker_name: string;
  similarity: number;
  confidence: 'high' | 'medium' | 'low';
}

// Segment types
export interface WordTimestamp {
  word: string;
  start_ms: number;
  end_ms: number;
  confidence?: number;
}

export interface Segment {
  id: string;
  recording_id: string;
  speaker_id?: string;
  temp_speaker_id?: string;
  text: string;
  start_ms: number;
  end_ms: number;
  confidence?: number;
  speaker_confidence?: number;
  is_user_verified: boolean;
  word_timestamps?: WordTimestamp[];
  speaker?: Speaker;
  created_at: string;
}

// API types
export interface SpeakerLabelRequest {
  segment_ids: string[];
  temp_speaker_id: string;
  speaker_id?: string;
  new_speaker_name?: string;
  apply_to_recording?: boolean;
}

export interface ExportRequest {
  format: 'txt' | 'srt' | 'vtt' | 'json';
  include_timestamps?: boolean;
  include_speaker_names?: boolean;
}

// Utility types
export type ExportFormat = 'txt' | 'srt' | 'vtt' | 'json';
