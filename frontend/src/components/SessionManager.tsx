import React, { useState, useEffect } from 'react';
import { sessionsAPI } from '../services/api';
import type { Session } from '../types';
import './SessionManager.css';

interface SessionManagerProps {
  onSessionSelect?: (sessionId: number | null) => void;
  currentSessionId?: number | null;
  onSessionCreated?: (sessionId: number) => void;
  onViewSession?: (sessionId: number) => void;
  onEndSession?: (sessionId: number) => void;
}

export const SessionManager: React.FC<SessionManagerProps> = ({
                                                                onSessionSelect,
                                                                currentSessionId,
                                                                onSessionCreated,
                                                                onViewSession,
                                                                onEndSession,
                                                              }) => {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [notes, setNotes] = useState('');

  // пагинация
  const [currentPage, setCurrentPage] = useState(1);
  const sessionsPerPage = 5;

  const loadSessions = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await sessionsAPI.getSessions();
      setSessions(Array.isArray(data) ? data : []);
      setCurrentPage(1); // Сброс на первую страницу при обновлении
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить сеансы');
      setSessions([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadSessions();
  }, []);

  const formatDateTime = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleString('ru-RU', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        timeZone: 'Europe/Moscow',
      });
    } catch {
      return dateString;
    }
  };

  const handleCreateSession = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const sessionNotes = notes.trim() || `Поездка: ${formatDateTime(new Date().toISOString())}`;

    try {
      const newSession = await sessionsAPI.createSession(sessionNotes);
      setSessions([newSession, ...sessions]);
      setNotes('');
      setShowCreateForm(false);
      onSessionSelect?.(newSession.id);
      onSessionCreated?.(newSession.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось создать сеанс');
    }
  };

  const handleEndSession = async (sessionId: number) => {
    console.log('handleEndSession called', { sessionId, currentSessionId, hasOnEndSession: !!onEndSession });

    if (!confirm('Вы уверены, что хотите завершить эту сессию и выключить камеру?')) {
      return;
    }

    setError(null);
    try {
      if (currentSessionId === sessionId && onEndSession) {
        console.log('Using onEndSession prop');
        await onEndSession(sessionId);
        await loadSessions();
      } else {
        console.log('Using API directly');
        await sessionsAPI.endSession(sessionId);
        await loadSessions();
      }
      console.log('Session ended successfully');
    } catch (err) {
      console.error('Error ending session:', err);
      setError(err instanceof Error ? err.message : 'Не удалось завершить сеанс');
    }
  };

  const handleDeleteSession = async (sessionId: number) => {
    if (!confirm('Вы уверены, что хотите удалить этот сеанс? Это действие нельзя отменить.')) {
      return;
    }

    setError(null);
    try {
      await sessionsAPI.deleteSession(sessionId);
      await loadSessions();
      setSessions(sessions.filter(session => session.id !== sessionId));
      if (currentSessionId === sessionId && onSessionSelect) {
        onSessionSelect(null);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось удалить сеанс');
      await loadSessions();
    }
  };

  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleString('ru-RU', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        timeZone: 'Europe/Moscow',
      });
    } catch {
      return dateString;
    }
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

  // пагинация
  const totalPages = Math.ceil(sessions.length / sessionsPerPage);
  const startIndex = (currentPage - 1) * sessionsPerPage;
  const currentSessions = sessions.slice(startIndex, startIndex + sessionsPerPage);

  const handleNextPage = () => {
    if (currentPage < totalPages) {
      setCurrentPage(currentPage + 1);
    }
  };

  const handlePrevPage = () => {
    if (currentPage > 1) {
      setCurrentPage(currentPage - 1);
    }
  };

  const handleFirstPage = () => {
    setCurrentPage(1);
  };

  return (
      <div className="sessions-manager">
        <div className="section-header">
          <h2 className="section-title">Поездки</h2>
          <button
              onClick={() => setShowCreateForm(!showCreateForm)}
              className="btn-add"
          >
            {showCreateForm ? 'Назад' : '+ Добавить'}
          </button>
        </div>

        {showCreateForm && (
            <form onSubmit={handleCreateSession} className="session-form">
              <div className="form-group">
                <label htmlFor="notes">Название (необязательно):</label>
                <textarea
                    id="notes"
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    placeholder="Добавьте информацию о вашей поездке..."
                    style={{ fontFamily: "inherit" }}
                    rows={3}
                />
              </div>
              <button type="submit" className="btn btn-primary">
                Создать сеанс
              </button>
            </form>
        )}

        {error && (
            <div className="error-message">
              {error}
              <button onClick={() => setError(null)} className="btn-close">×</button>
            </div>
        )}

        {loading ? (
            <div className="loading">Загрузка сеансов...</div>
        ) : !sessions || sessions.length === 0 ? (
            <div className="empty-state">
              <p>Нажмите "Добавить", чтобы начать первую поездку.</p>
            </div>
        ) : (
            <>
              <div className="session-list">
                {currentSessions.map((session) => (
                    <div className="session-card">
                        <div className="session-header">
                            <div>
                                <div className="session-time">Поездка: {formatDateTime(session.start_time)}</div>
                                <div className="session-meta">
                                    <div className="session-meta-item">Начало: {formatDate(session.start_time)}</div>
                                    {session.end_time && (
                                        <div className="session-meta-item">Конец: {formatDate(session.end_time)}</div>
                                    )}
                                    <div className="session-meta-item">Время: {getDuration(session.start_time, session.end_time)}</div>
                                </div>
                            </div>
                            <div className="status-badge">{session.status === 'completed' || session.status === 'завершен' ? 'Завершен' : 'Активен'}</div>
                        </div>
                        {(session.status === 'completed' || session.status === 'завершен') && (
                            <div className="session-actions">
                                <button className="btn-small btn-view" onClick={() => onViewSession?.(session.id)}>
                                    Смотреть
                                </button>
                                <button className="btn-small btn-delete" onClick={() => handleDeleteSession(session.id)}>
                                    🗑️
                                </button>
                            </div>
                        )}
                        {session.status === 'active' && (
                            <div className="session-actions">
                                <button className="btn-small btn-view" onClick={() => onSessionSelect?.(session.id)}>
                                    Выбрать
                                </button>
                                <button className="btn-small btn-delete" onClick={() => handleEndSession(session.id)}>
                                    Конец
                                </button>
                            </div>
                        )}
                    </div>
                ))}
              </div>
              <div className="pagination">
                {currentPage > 1 && (
                    <>
                      <button
                          onClick={handleFirstPage}
                          className="btn btn-primary btn-pagination"
                      >
                        В начало
                      </button>
                      <button
                          onClick={handlePrevPage}
                          className="btn btn-secondary btn-pagination"
                      >
                        ← Предыдущая
                      </button>
                    </>
                )}
                <span className="pagination-info">
                Страница {currentPage} из {totalPages}
                </span>
                {currentPage < totalPages && (
                    <button
                        onClick={handleNextPage}
                        className="btn btn-primary btn-pagination"
                    >
                      Следующая →
                    </button>
                )}
              </div>
            </>
        )}
      </div>
  );
};
