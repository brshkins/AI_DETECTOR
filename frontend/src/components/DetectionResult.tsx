import React from 'react';
import type { DetectionResult as DetectionResultType } from '../types';
import './DetectionResult.css';

interface DetectionResultProps {
  result: DetectionResultType | null;
}

export const DetectionResult: React.FC<DetectionResultProps> = ({ result }) => {
  if (!result) {
    return (
      <div className="detection-result">
        <div className="detection-placeholder">
          <p>Дождитесь результатов детекции...</p>
        </div>
      </div>
    );
  }

  const getAlertColor = (alertLevel: string) => {
    switch (alertLevel.toLowerCase()) {
      case 'high':
        return '#ef4444';
      case 'medium':
        return '#f59e0b';
      case 'low':
        return '#10b981';
      default:
        return '#6b7280';
    }
  };

  const getScoreColor = (score: number) => {
    if (score >= 0.7) return '#ef4444';
    if (score >= 0.4) return '#f59e0b';
    return '#10b981';
  };

  return (
    <div className="detection-result">
      <div className="detection-header">
        <h3>Результаты</h3>
        <span
          className="alert-badge"
          style={{ backgroundColor: getAlertColor(result.alert_level) }}
        >
          {result.alert_level.toUpperCase()}
        </span>
      </div>

      <div className="detection-content">
        <div className="detection-item">
          <span className="detection-label">Статус:</span>
          <span
            className={`detection-value ${result.is_drowsy ? 'drowsy' : 'alert'}`}
          >
            {result.is_drowsy ? '⚠️ Обнаружена сонливость' : '✅ Стабильное состояние'}
          </span>
        </div>

        <div className="detection-item">
          <span className="detection-label">Процент сонливости:</span>
          <span
            className="detection-value score"
            style={{ color: getScoreColor(result.drowsiness_score) }}
          >
            {(result.drowsiness_score * 100).toFixed(1)}%
          </span>
        </div>

        <div className="detection-item">
          <span className="detection-label">Уровень тревоги:</span>
          <span className="detection-value">{result.alert_level}</span>
        </div>
      </div>

      <div className="score-bar-container">
        <div className="score-bar-label">Шкала сонливости</div>
        <div className="score-bar">
          <div
            className="score-bar-fill"
            style={{
              width: `${result.drowsiness_score * 100}%`,
              backgroundColor: getScoreColor(result.drowsiness_score),
            }}
          />
        </div>
      </div>
    </div>
  );
};

