import React, { useState, useEffect, useCallback } from 'react';
import { eventsAPI } from '../services/api';
import type { Event } from '../types';
import './EventHistory.css';

interface EventHistoryProps {
  sessionId: number | null;
}

const EVENTS_PER_PAGE = 3;

export const EventHistory: React.FC<EventHistoryProps> = ({ sessionId }) => {
  const [events, setEvents] = useState<Event[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<'all' | 'drowsy' | 'alert'>('all');
  const [visibleCount, setVisibleCount] = useState(EVENTS_PER_PAGE);

  useEffect(() => {
    if (sessionId) {
      loadEvents();
    } else {
      setEvents([]);
    }
  }, [sessionId]);

  const loadEvents = async () => {
    if (!sessionId) return;

    setLoading(true);
    setError(null);
    try {
      const data = await eventsAPI.getEvents(sessionId);
      setEvents(Array.isArray(data) ? data : []);
      setVisibleCount(EVENTS_PER_PAGE); // Сброс при обновлении
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить события');
      setEvents([]);
    } finally {
      setLoading(false);
    }
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

  const combinedEvents = combineConsecutiveEvents(filteredEvents);
  const visibleEvents = combinedEvents.slice(0, visibleCount);

  const loadMore = () => {
    setVisibleCount(prev => prev + EVENTS_PER_PAGE);
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const getScoreColor = (score: number) => {
    if (score >= 0.7) return '#b54562';
    if (score >= 0.4) return '#e8aa67';
    return '#71cdb2';
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
  };

  if (!sessionId) {
    return (
        <div className="event-history">
          <div className="empty-state">
            <p>Выберите сеанс для просмотра истории</p>
          </div>
        </div>
    );
  }

  return (
      <div className="event-history">
        <div className="event-header">
          <h2>История поездки</h2>
          <button onClick={loadEvents} className="btn btn-secondary btn-sm" disabled={loading}>
            {loading ? 'Загрузка...' : 'Обновить'}
          </button>
        </div>

        {error && (
            <div className="error-message">
              {error}
              <button onClick={() => setError(null)} className="btn-close">×</button>
            </div>
        )}

        {loading && combinedEvents.length === 0 ? (
            <div className="loading">Загрузка поездок...</div>
        ) : (
            <>
              <div className="event-stats">
                <div className="stat-item">
                  <span className="stat-label">Всего</span>
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
                  <span className="stat-label">Средняя оценка</span>
                  <span className="stat-value">
                {(stats.avgScore * 100).toFixed(1)}%
              </span>
                </div>
              </div>

              <div className="event-filters">
                <button onClick={() => setFilter('all')} className={`filter-btn ${filter === 'all' ? 'active' : ''}`}>
                  Все
                </button>
                <button onClick={() => setFilter('drowsy')} className={`filter-btn ${filter === 'drowsy' ? 'active' : ''}`}>
                  Сонливость
                </button>
                <button onClick={() => setFilter('alert')} className={`filter-btn ${filter === 'alert' ? 'active' : ''}`}>
                  Бодрствование
                </button>
              </div>

              {visibleEvents.length === 0 ? (
                  <div className="empty-state">
                    <p>Состояние не найдено.</p>
                  </div>
              ) : (
                  <div className="event-list" style={{ maxHeight: '400px', overflowY: 'auto' }}>
                    {visibleEvents.map((event, index) => {
                      const avgScore = event.drowsiness_score;
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
                          {event.is_drowsy ? 'Сонливость обнаружена' : 'Бодрствование'}
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
                                {event.count && event.count > 1 && (
                                    <span className="event-count">
                            ({event.count} записей, {durationMinutes}м {durationSeconds}с)
                          </span>
                                )}
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

                    {/* Кнопка "Загрузить ещё" */}
                    {visibleCount < combinedEvents.length && (
                        <div className="load-more-container">
                          <button onClick={loadMore} className="btn btn-secondary">
                            Загрузить ещё ({Math.min(EVENTS_PER_PAGE, combinedEvents.length - visibleCount)})
                          </button>
                        </div>
                    )}
                  </div>
              )}
            </>
        )}
      </div>
  );
};