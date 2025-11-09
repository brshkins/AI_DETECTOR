import React, { useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { CameraFeed, type CameraFeedRef } from '../components/CameraFeed';
import { DetectionResult } from '../components/DetectionResult';
import { SessionManager } from '../components/SessionManager';
import { Navbar } from '../components/Navbar';
import { eventsAPI, sessionsAPI } from '../services/api';
import type { DetectionResult as DetectionResultType } from '../types';
import './HomePage.css';

export const HomePage: React.FC = () => {
  const navigate = useNavigate();
  const [currentSessionId, setCurrentSessionId] = useState<number | null>(null);
  const [detectionResult, setDetectionResult] = useState<DetectionResultType | null>(null);
  const cameraStartRef = useRef<CameraFeedRef>(null);

  const handleSessionCreated = (sessionId: number) => {
    setCurrentSessionId(sessionId);
    setTimeout(() => {
      cameraStartRef.current?.startCamera();
    }, 100);
  };

  const handleViewSession = (sessionId: number) => {
    navigate(`/session/${sessionId}`);
  };

  const handleEndSession = async () => {
    if (!currentSessionId) return;
    cameraStartRef.current?.stopCamera();
    
    try {
      await sessionsAPI.endSession(currentSessionId);
      setCurrentSessionId(null);
      setDetectionResult(null);

    } catch (error) {
      console.error('Failed to end session:', error);
    }
  };

  const handleEndSessionFromManager = async (sessionId: number) => {
    if (sessionId !== currentSessionId) {
      return;
    }
    await handleEndSession();
  };

  const handleDetectionResult = async (result: DetectionResultType) => {
    setDetectionResult(result);

    if (currentSessionId && result) {
      try {
        await eventsAPI.saveEvent({
          session_id: currentSessionId,
          drowsiness_score: result.drowsiness_score,
          is_drowsy: result.is_drowsy,
        });
      } catch (error) {
        console.error('Failed to save event:', error);
      }
    }
  };

  return (
    <div className="home-page">
      <Navbar />
      <div className="page-header">
        <h1>Детектор сонливости</h1>
        <p>Определение вашей сонливости в реальном времени</p>
      </div>

      <div className="page-content">
        <div className="main-section">
          <div className="camera-section">
            <CameraFeed
              ref={cameraStartRef}
              onDetectionResult={handleDetectionResult}
              captureInterval={1000}
              sessionId={currentSessionId || undefined}
              hasActiveSession={!!currentSessionId}
              onEndSession={handleEndSession}
            />
          </div>

          <div className="results-section">
            <DetectionResult result={detectionResult} />
          </div>
        </div>

        <div className="sidebar-section">
          <div className="sidebar-item">
            <SessionManager
              onSessionSelect={setCurrentSessionId}
              currentSessionId={currentSessionId}
              onSessionCreated={handleSessionCreated}
              onViewSession={handleViewSession}
              onEndSession={handleEndSessionFromManager}
            />
          </div>
        </div>
      </div>
    </div>
  );
};

