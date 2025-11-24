import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { SessionManager } from '../components/SessionManager';
import { Navbar } from '../components/Navbar';
import { DecorativeElements } from '../components/DecorativeElements';
import './HomePage.css';

export const HomePage: React.FC = () => {
    const navigate = useNavigate();
    const [currentSessionId, setCurrentSessionId] = useState<number | null>(null);

    const handleSessionCreated = (sessionId: number) => {
        navigate(`/detection/${sessionId}`);
    };

    const handleViewSession = (sessionId: number) => {
        navigate(`/session/${sessionId}`);
    };

    return (
        <div className="home-page">
            <DecorativeElements />
            <Navbar />
            <div className="page-header">
                <h1>Детектор сонливости</h1>
                <p>Управляйте своими поездками</p>
            </div>

            <div className="dashboard-content">
                <div className="sessions-manager">
                    <SessionManager
                        onSessionSelect={setCurrentSessionId}
                        currentSessionId={currentSessionId}
                        onSessionCreated={handleSessionCreated}
                        onViewSession={handleViewSession}
                        onEndSession={() => {}}
                    />
                </div>
            </div>
        </div>
    );
};