import React, { useState, useRef, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { CameraFeed, type CameraFeedRef } from '../components/CameraFeed';
import { DetectionResult } from '../components/DetectionResult';
import { eventsAPI, sessionsAPI } from '../services/api';
import type { DetectionResultType } from '../types';
import { Navbar } from '../components/Navbar';
import { DecorativeElements } from '../components/DecorativeElements';
import './DetectionPage.css';

export const DetectionPage: React.FC = () => {
    const { sessionId } = useParams<{ sessionId: string }>();
    const navigate = useNavigate();
    const sessionID = parseInt(sessionId || '', 10);
    const [detectionResult, setDetectionResult] = useState<DetectionResultType | null>(null);
    const cameraFeedRef = useRef<CameraFeedRef>(null);

    const handleEndSession = async () => {
        if (isNaN(sessionID)) return;

        cameraFeedRef.current?.stopCamera();
        try {
            await sessionsAPI.endSession(sessionID);
        } catch (error) {
            console.error('Failed to end session:', error);
        } finally {
            navigate('/');
        }
    };

    const handleDetectionResult = async (result: DetectionResultType) => {
        setDetectionResult(result);

        try {
            await eventsAPI.saveEvent({
                session_id: sessionID,
                drowsiness_score: result.drowsiness_score,
                is_drowsy: result.is_drowsy,
            });
        } catch (error) {
            console.error('Failed to save event:', error);
        }
    };

    useEffect(() => {
        if (isNaN(sessionID)) {
            navigate('/');
            return;
        }

        if (cameraFeedRef.current) {
            cameraFeedRef.current.startCamera();
        }

        return () => {
            if (cameraFeedRef.current) {
                cameraFeedRef.current.stopCamera();
            }
        };
    }, [sessionID, navigate]);

    if (isNaN(sessionID)) {
        return null;
    }

    return (
        <div className="detection-page">
            <DecorativeElements />
            <Navbar />
            <div className="page-header">
                <h1>Детектор сонливости</h1>
                <p>Режим вождения: активен</p>
            </div>
            <div className="detection-layout">
                <div className="camera-column">
                    <CameraFeed
                        ref={cameraFeedRef}
                        onDetectionResult={handleDetectionResult}
                        captureInterval={1000}
                        hasActiveSession={true}
                        onEndSession={handleEndSession}
                    />
                </div>
                <div className="results-column">
                    <DetectionResult result={detectionResult} />
                    <button className="end-session-button" onClick={handleEndSession}>
                        Завершить поездку
                    </button>
                </div>
            </div>
        </div>
    );
};