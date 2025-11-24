import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { sessionsAPI, eventsAPI } from '../services/api';
import type { Session, Event } from '../types';
import { Navbar } from '../components/Navbar';
import { DecorativeElements } from '../components/DecorativeElements';
import './SessionDetailsPage.css';

export const SessionDetailsPage: React.FC = () => {
    const { sessionId } = useParams<{ sessionId: string }>();
    const navigate = useNavigate();
    const [session, setSession] = useState<Session | null>(null);
    const [events, setEvents] = useState<Event[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [filter, setFilter] = useState<'all' | 'drowsy' | 'alert'>('all');

    useEffect(() => {
        if (!sessionId) {
            navigate('/');
            return;
        }

        const loadData = async () => {
            setLoading(true);
            setError(null);
            try {
                const id = parseInt(sessionId, 10);
                const [sessionsData, eventsData] = await Promise.all([
                    sessionsAPI.getSessions(),
                    eventsAPI.getEvents(id),
                ]);

                const foundSession = sessionsData.find((s) => s.id === id);
                if (!foundSession) {
                    setError('Сеанс не найден');
                    return;
                }

                setSession(foundSession);
                setEvents(Array.isArray(eventsData) ? eventsData : []);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Не удалось загрузить данные');
            } finally {
                setLoading(false);
            }
        };

        loadData();
    }, [sessionId, navigate]);

    const countStateChanges = (events: Event[]): number => {
        if (events.length < 2) return 0;
        let changes = 0;
        for (let i = 1; i < events.length; i++) {
            if (events[i - 1].is_drowsy !== events[i].is_drowsy) {
                changes++;
            }
        }
        return changes;
    };

    const combineConsecutiveEvents = (events: Event[]): Array<Event & { duration?: number; count?: number }> => {
        if (events.length === 0) return [];

        const combined: Array<Event & { duration?: number; count?: number }> = [];
        let currentGroup: Event[] = [events[0]];

        for (let i = 1; i < events.length; i++) {
            const prevEvent = events[i - 1];
            const currentEvent = events[i];

            if (prevEvent.is_drowsy === currentEvent.is_drowsy) {
                currentGroup.push(currentEvent);
            } else {
                const mostRecentTime = new Date(currentGroup[0].timestamp).getTime();
                const oldestTime = new Date(currentGroup[currentGroup.length - 1].timestamp).getTime();
                const duration = Math.floor((mostRecentTime - oldestTime) / 1000);

                combined.push({
                    ...currentGroup[0],
                    duration,
                    count: currentGroup.length,
                });

                currentGroup = [currentEvent];
            }
        }

        if (currentGroup.length > 0) {
            const mostRecentTime = new Date(currentGroup[0].timestamp).getTime();
            const oldestTime = new Date(currentGroup[currentGroup.length - 1].timestamp).getTime();
            const duration = Math.floor((mostRecentTime - oldestTime) / 1000);

            combined.push({
                ...currentGroup[0],
                duration,
                count: currentGroup.length,
            });
        }

        return combined;
    };

    const filteredEvents = (events || []).filter((event) => {
        if (filter === 'drowsy') return event.is_drowsy;
        if (filter === 'alert') return !event.is_drowsy;
        return true;
    });

    // Группировка только если filter === 'all'
    const displayEvents = filter === 'all'
        ? combineConsecutiveEvents(filteredEvents)
        : filteredEvents; // Для drowsy и alert без группировки

    const stateChanges = countStateChanges(events || []);

    const formatDate = (dateString: string) => {
        return new Date(dateString).toLocaleString();
    };

    const getDuration = (startTime: string, endTime?: string) => {
        const start = new Date(startTime);
        const end = endTime ? new Date(endTime) : new Date();
        const diff = Math.floor((end.getTime() - start.getTime()) / 1000);
        const hours = Math.floor(diff / 3600);
        const minutes = Math.floor((diff % 3600) / 60);
        const seconds = diff % 60;
        return `${hours}ч ${minutes}м ${seconds}с`;
    };

    const getScoreColor = (score: number) => {
        if (score >= 0.7) return '#ef4444';
        if (score >= 0.4) return '#f59e0b';
        return '#10b981';
    };

    const safeEvents = events || [];
    const stats = {
        total: safeEvents.length,
        drowsy: safeEvents.filter((e) => e.is_drowsy).length,
        alert: safeEvents.filter((e) => !e.is_drowsy).length,
        avgScore:
            safeEvents.length > 0
                ? safeEvents.reduce((sum, e) => sum + e.drowsiness_score, 0) / safeEvents.length
                : 0,
        stateChanges,
    };

    if (loading) {
        return (
            <div className="session-details-page">
                <Navbar />
                <div className="loading-container">
                    <div className="loading">Загрузка...</div>
                </div>
            </div>
        );
    }

    if (error || !session) {
        return (
            <div className="session-details-page">
                <Navbar />
                <div className="error-container">
                    <p>{error || 'Сеанс не найден'}</p>
                    <button onClick={() => navigate('/')} className="btn btn-primary">
                        Вернуться на главную
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="session-details-page">
            <DecorativeElements />
            <Navbar />
            <div className="session-details-container">
                <div className="session-details-header">
                    <button onClick={() => navigate('/')} className="btn btn-secondary">
                        ← Назад
                    </button>
                    <h1>{session.notes || `Детали сеанса #${session.id}`}</h1>
                </div>

                <div className="session-info-card">
                    <h2>Информация о поездке</h2>
                    <div className="session-info-grid">
                        <div className="info-item">
                            <span className="info-label">Начало:</span>
                            <span className="info-value">{formatDate(session.start_time)}</span>
                        </div>
                        {session.end_time && (
                            <div className="info-item">
                                <span className="info-label">Конец:</span>
                                <span className="info-value">{formatDate(session.end_time)}</span>
                            </div>
                        )}
                        <div className="info-item">
                            <span className="info-label">Длительность:</span>
                            <span className="info-value">{getDuration(session.start_time, session.end_time)}</span>
                        </div>
                        <div className="info-item">
                            <span className="info-label">Статус:</span>
                            <span className={`status-badge ${session.status}`}>
                {session.status === 'completed' || session.status === 'завершен' ? 'завершен' : session.status}
              </span>
                        </div>
                    </div>
                </div>

                <div className="session-stats-card">
                    <h2>Статистика</h2>
                    <div className="stats-grid">
                        <div className="stat-item">
                            <span className="stat-label">Всего детекций</span>
                            <span className="stat-value">{stats.total}</span>
                        </div>
                        <div className="stat-item">
                            <span className="stat-label">Сонливость</span>
                            <span className="stat-value drowsy">{stats.drowsy}</span>
                        </div>
                        <div className="stat-item">
                            <span className="stat-label">Бодрствование</span>
                            <span className="stat-value alert">{stats.alert}</span>
                        </div>
                        <div className="stat-item">
                            <span className="stat-label">Смена состояний</span>
                            <span className="stat-value">{stats.stateChanges}</span>
                        </div>
                        <div className="stat-item">
                            <span className="stat-label">Средняя оценка</span>
                            <span className="stat-value">
                {(stats.avgScore * 100).toFixed(1)}%
              </span>
                        </div>
                    </div>
                </div>

                <div className="event-history-card">
                    <div className="event-history-header">
                        <h2>Поездка</h2>
                        <div className="event-filters">
                            <button
                                onClick={() => setFilter('all')}
                                className={`filter-btn ${filter === 'all' ? 'active' : ''}`}
                            >
                                Все
                            </button>
                            <button
                                onClick={() => setFilter('drowsy')}
                                className={`filter-btn ${filter === 'drowsy' ? 'active' : ''}`}
                            >
                                Сонливость
                            </button>
                            <button
                                onClick={() => setFilter('alert')}
                                className={`filter-btn ${filter === 'alert' ? 'active' : ''}`}
                            >
                                Бодрствование
                            </button>
                        </div>
                    </div>

                    {displayEvents.length === 0 ? (
                        <div className="empty-state">
                            <p>События не найдены</p>
                        </div>
                    ) : (
                        <div className="event-list">
                            {displayEvents.map((event, index) => {
                                const avgScore = 'drowsiness_score' in event ? event.drowsiness_score : 0;
                                const durationMinutes = event.duration ? Math.floor(event.duration / 60) : 0;
                                const durationSeconds = event.duration ? event.duration % 60 : 0;

                                return (
                                    <div
                                        key={`${event.id}-${index}`}
                                        className={`event-item ${event.is_drowsy ? 'drowsy' : 'alert'}`}
                                    >
                                        <div className="event-icon">
                                            {event.is_drowsy ? '✴️' : '✳️'}
                                        </div>
                                        <div className="event-content">
                                            <div className="event-header-row">
                        <span className="event-status">
                          {event.is_drowsy ? 'Сонливость' : 'Бодрствование'}
                        </span>
                                                <span className="event-time">{formatDate(event.timestamp)}</span>
                                            </div>
                                            <div className="event-score-row">
                                                <span className="event-label">Уровень:</span>
                                                <span
                                                    className="event-score"
                                                    style={{ color: getScoreColor(avgScore) }}
                                                >
                          {(avgScore * 100).toFixed(1)}%
                        </span>
                                            </div>
                                        </div>
                                        <div
                                            className="event-score-bar"
                                            style={{
                                                width: `${avgScore * 100}%`,
                                                backgroundColor: getScoreColor(avgScore),
                                            }}
                                        />
                                    </div>
                                );
                            })}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};