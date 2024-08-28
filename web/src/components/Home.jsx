import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import '../App.css';
const BaseURL = 'https://dsandbox.online'
const Home = () => {
    const [url, setURL] = useState('');
    const [shortID, setShortID] = useState('');
    const [result, setResult] = useState(null); // To store the result of the URL shortening
    const [error, setError] = useState('');
    const navigate = useNavigate();

    const handleSubmit = async (event) => {
        event.preventDefault();
        setError('');
        try {
            new URL(url); // Will throw if the URL is invalid
        } catch (error) {
            setError('Invalid URL format');
            return;
        }
        try {
            const response = await fetch(`${BaseURL}/api/core/create`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ original_url: url, short_id: shortID })
            });
            if (!response.ok) throw new Error('Failed to create short URL');
            const data = await response.json();
            setResult({
                shortID: data.short_id,
                shortURL: `${BaseURL}/s/${data.short_id}`
            });
        } catch (error) {
            setError(error.message);
        }
    };

    const handleGoToAnalytics = () => {
        if (result && result.shortID) {
            navigate(`/stats/${result.shortID}`);
        }
    };

    return (
        <div className="container">
            <h1>URL Shortener</h1>
            <form onSubmit={handleSubmit}>
                <input
                    type="text"
                    value={url}
                    onChange={(e) => setURL(e.target.value)}
                    placeholder="Enter URL"
                />
                <input
                    type="text"
                    value={shortID}
                    onChange={(e) => setShortID(e.target.value)}
                    placeholder="Custom Short ID (optional)"
                />
                <button type="submit">Submit</button>
            </form>
            {error && <p className="error">{error}</p>}
            {result && (
                <div className="result">
                    <p>Shortened URL: <a href={result.shortURL} target="_blank" rel="noopener noreferrer">{result.shortURL}</a></p>
                    <button onClick={handleGoToAnalytics}>Go to Analytics</button>
                </div>
            )}
        </div>
    );
};

export default Home;
