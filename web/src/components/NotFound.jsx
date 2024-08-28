import React from 'react';
import { Link } from 'react-router-dom';
import '../App.css';

const NotFound = () => {
  return (
    <div className="container">
      <h2>404 - Page Not Found</h2>
      <Link to="/">Back to Home</Link>
    </div>
  );
};

export default NotFound;
