export interface User {
  id: number;
  email: string;
  username: string;
  created_at: string;
}

export interface Session {
  id: number;
  user_id: number;
  start_time: string;
  end_time?: string;
  status: string;
  notes?: string;
}

export interface Event {
  id: number;
  session_id: number;
  drowsiness_score: number;
  is_drowsy: boolean;
  timestamp: string;
}

export interface DetectionResult {
  is_drowsy: boolean;
  drowsiness_score: number;
  alert_level: string;
  timestamp: number;
  inference_time?: number;
}

export interface CreateSessionRequest {
  notes?: string;
}

export interface CreateEventRequest {
  session_id: number;
  drowsiness_score: number;
  is_drowsy: boolean;
}

