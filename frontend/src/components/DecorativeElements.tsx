import React from 'react';
import './DecorativeElements.css';

export const DecorativeElements: React.FC = () => {
  return (
      <div className="decorative-elements">
          <div className="star star-1">⭐</div>
          <div className="star star-2">✨</div>
          <div className="star star-3">⭐</div>
          <div className="star star-4">✨</div>
          <div className="star star-5">⭐</div>
          <div className="star star-6">✨</div>

          <div className="cloud cloud-1">☁️</div>
          <div className="cloud cloud-2">☁️</div>
          <div className="cloud cloud-3">☁️</div>

          <div className="bubble bubble-1"></div>
          <div className="bubble bubble-2"></div>
          <div className="bubble bubble-3"></div>
          <div className="bubble bubble-4"></div>
          <div className="bubble bubble-5"></div>
          <div className="bubble bubble-6"></div>

          <svg className="wave-line wave-1" viewBox="0 0 200 20" preserveAspectRatio="none">
              <path d="M0,10 Q50,0 100,10 T200,10" stroke="rgba(168, 213, 226, 0.3)" strokeWidth="2" fill="none"/>
          </svg>
          <svg className="wave-line wave-2" viewBox="0 0 200 20" preserveAspectRatio="none">
              <path d="M0,10 Q50,20 100,10 T200,10" stroke="rgba(255, 182, 193, 0.3)" strokeWidth="2" fill="none"/>
          </svg>
      </div>
  );
};


