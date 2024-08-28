import React from 'react';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import './App.css';
import Home from './components/Home';
import Analytics from './components/Analytics';
import NotFound from './components/NotFound';

function App() {
  return (
    <div className="App">
      <main>
        <Router>
          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/stats/:shortID" element={<Analytics />} />
            <Route path="*" element={<NotFound />} />
          </Routes>
        </Router>
      </main>
      <footer>
        <p>&copy; 2024 URL Redirect App</p>
      </footer>
    </div>
  );
}

export default App;
