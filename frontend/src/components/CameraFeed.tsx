import { useRef, useEffect, useState, useCallback, useImperativeHandle, forwardRef } from 'react';
import { wsService } from '../services/websocket';
import type { DetectionResult } from '../types';
import './CameraFeed.css';

interface CameraFeedProps {
    onDetectionResult?: (result: DetectionResult) => void;
    onError?: (error: string) => void;
    captureInterval?: number;
    sessionId?: string | number;
    hasActiveSession?: boolean;
    onEndSession?: () => void;
}

export interface CameraFeedRef {
    startCamera: () => void;
    stopCamera: () => void;
}

export const CameraFeed = forwardRef<CameraFeedRef, CameraFeedProps>(
    ({ onDetectionResult, onError, captureInterval = 1000, hasActiveSession = false, onEndSession }, ref) => {
        // Убрали `sessionId` из деструктуризации — он не используется
        const videoRef = useRef<HTMLVideoElement>(null);
        const canvasRef = useRef<HTMLCanvasElement>(null);
        const [isStreaming, setIsStreaming] = useState(false);
        const [isDetecting, setIsDetecting] = useState(false);
        const [hasDetected, setHasDetected] = useState(false);
        const [error, setError] = useState<string | null>(null);
        const streamRef = useRef<MediaStream | null>(null);
        const sequenceNumberRef = useRef<number>(0);

        const startCamera = useCallback(async () => {
            if (!hasActiveSession) {
                console.log('Cannot start camera: no active session');
                return;
            }

            console.log('Starting camera...');
            try {
                const stream = await navigator.mediaDevices.getUserMedia({
                    video: { width: { ideal: 640 }, height: { ideal: 480 }, facingMode: 'user' },
                    audio: false,
                });

                if (videoRef.current) {
                    videoRef.current.srcObject = stream;
                    streamRef.current = stream;
                    setIsStreaming(true);
                    setHasDetected(false);
                    setError(null);
                    console.log('Camera started successfully');
                } else {
                    console.error('Video ref is null');
                }
            } catch (err) {
                const errorMessage = err instanceof Error ? err.message : 'Не удалось получить доступ к камере';
                console.error('Failed to start camera:', err);
                setError(errorMessage);
                onError?.(errorMessage);
            }
        }, [hasActiveSession, onError]);

        useImperativeHandle(ref, () => ({
            startCamera,
            stopCamera,
        }));

        const stopCamera = useCallback(() => {
            if (streamRef.current) {
                streamRef.current.getTracks().forEach(track => track.stop());
                streamRef.current = null;
            }
            if (videoRef.current) {
                videoRef.current.srcObject = null;
            }
            setIsStreaming(false);
            setHasDetected(false);
        }, []);

        const handleEndSession = useCallback(() => {
            stopCamera();
            onEndSession?.();
        }, [stopCamera, onEndSession]);

        const captureFrame = useCallback(() => {
            if (!videoRef.current || !canvasRef.current || isDetecting || !isStreaming) {
                console.log('captureFrame skipped:', { 
                    hasVideo: !!videoRef.current, 
                    hasCanvas: !!canvasRef.current, 
                    isDetecting, 
                    isStreaming 
                });
                return;
            }

            const video = videoRef.current;
            const canvas = canvasRef.current;

            if (video.videoWidth === 0 || video.videoHeight === 0) {
                console.log('Video not ready:', { videoWidth: video.videoWidth, videoHeight: video.videoHeight });
                return;
            }

            if (!wsService.isConnected()) {
                console.warn('WebSocket not connected, cannot send frame');
                return;
            }

            canvas.width = video.videoWidth;
            canvas.height = video.videoHeight;

            const ctx = canvas.getContext('2d');
            if (!ctx) {
                console.error('Failed to get canvas context');
                return;
            }

            ctx.drawImage(video, 0, 0);

            try {
                setIsDetecting(true);
                const base64 = canvas.toDataURL('image/jpeg', 0.8).split(',')[1];
                const timestamp = Date.now();
                const sequenceNumber = sequenceNumberRef.current++;

                console.log(`Sending frame ${sequenceNumber} to WebSocket`);
                wsService.send('FRAME', {
                    frame: base64,
                    timestamp,
                    sequenceNumber,
                });
            } catch (err) {
                const errorMessage = err instanceof Error ? err.message : 'Detection failed';
                console.error('Error capturing frame:', err);
                setError(errorMessage);
                onError?.(errorMessage);
            } finally {
                setIsDetecting(false);
            }
        }, [isDetecting, isStreaming, onError]);

        // Регистрируем обработчики сообщений один раз
        useEffect(() => {
            const handleDetectionResult = (payload: any) => {
                const result: DetectionResult = {
                    is_drowsy: payload.is_drowsy,
                    drowsiness_score: payload.drowsiness_score,
                    alert_level: payload.alert_level,
                    inference_time_ms: payload.inference_time,
                    timestamp: payload.timestamp,
                    sequence_number: payload.sequence_number, // ✅ Теперь тип поддерживает это поле
                };
                setHasDetected(true);
                onDetectionResult?.(result);
            };

            const handleError = (payload: any) => {
                const message = payload.message || 'Processing failed';
                setError(message);
                onError?.(message);
            };

            const handleWelcome = (payload: any) => {
                console.log('WebSocket welcome:', payload);
            };

            wsService.on('DETECTION_RESULT', handleDetectionResult);
            wsService.on('ERROR', handleError);
            wsService.on('WELCOME', handleWelcome);

            return () => {
                // НЕ отключаем WebSocket при размонтировании компонента
                // Только удаляем обработчики
                wsService.off('DETECTION_RESULT', handleDetectionResult);
                wsService.off('ERROR', handleError);
                wsService.off('WELCOME', handleWelcome);
            };
        }, [onDetectionResult, onError]); // Регистрируем обработчики один раз

        // Подключаемся к WebSocket когда появляется активная сессия
        useEffect(() => {
            if (hasActiveSession) {
                if (!wsService.isConnected()) {
                    console.log('Connecting WebSocket for active session...');
                    wsService.connect();
                } else {
                    console.log('WebSocket already connected');
                }
            } else {
                console.log('No active session, WebSocket connection skipped');
            }
        }, [hasActiveSession]); // Подключаемся при изменении hasActiveSession

        useEffect(() => {
            let intervalId: NodeJS.Timeout | null = null;
            if (isStreaming && captureInterval > 0) {
                intervalId = setInterval(captureFrame, captureInterval);
            }
            return () => {
                if (intervalId) clearInterval(intervalId);
            };
        }, [isStreaming, captureInterval, captureFrame]);

        useEffect(() => {
            return () => {
                stopCamera();
            };
        }, [stopCamera]);

        return (
            <div className="camera-feed">
                <div className="camera-container">
                    <video ref={videoRef} autoPlay playsInline muted className="camera-video" />
                    <canvas ref={canvasRef} style={{ display: 'none' }} />
                    {!isStreaming && (
                        <div className="camera-placeholder">
                            <p>Камера выключена</p>
                            <p className="camera-hint">Создайте сеанс, чтобы начать</p>
                        </div>
                    )}
                    {error && (
                        <div className="camera-error">
                            <p>Error: {error}</p>
                            <button onClick={() => setError(null)} className="btn btn-secondary">
                                Dismiss
                            </button>
                        </div>
                    )}
                </div>
                {isStreaming && (
                    <div className="camera-controls">
                        <button onClick={handleEndSession} className="btn btn-danger">
                            Завершить сеанс
                        </button>
                        <span className="status-indicator">{hasDetected ? 'Активен' : 'Запуск...'}</span>
                    </div>
                )}
            </div>
        );
    }
);

CameraFeed.displayName = 'CameraFeed';