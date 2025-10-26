package models

import "time"

type VideoFrame struct {
	Frame          string `json:"frame"`
	Timestamp      int64  `json:"timestamp"`
	SequenceNumber int32  `json:"sequence_number,omitempty"`
}

type DetectionResult struct {
	IsDrowsy           bool    `json:"is_drowsy"`
	DrowsinessScore    float32 `json:"drowsiness_score"`
	EyesLookingForward bool    `json:"eyes_looking_forward"`
	EyeDirectionScore  float32 `json:"eye_direction_score"`
	HeadAngle          float32 `json:"head_angle"`
	AlertLevel         string  `json:"alert_level"`
	InferenceTimeMs    float32 `json:"inference_time_ms"`
	Timestamp          int64   `json:"timestamp"`
	ClientTimestamp    int64   `json:"client_timestamp,omitempty"`
	SequenceNumber     int32   `json:"sequence_number,omitempty"`
}

type ErrorResponse struct {
	Error     string `json:"error"`
	Timestamp int64  `json:"timestamp"`
	Code      string `json:"code,omitempty"`
}

type HealthStatus struct {
	Status        string        `json:"status"`
	GoBackend     string        `json:"go_backend"`
	PythonService bool          `json:"python_service"`
	ActiveClients int           `json:"active_clients"`
	Uptime        time.Duration `json:"uptime"`
	Version       string        `json:"version,omitempty"`
}
