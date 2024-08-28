import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import '../App.css'; 
import { BaseURL } from '../env';

const Analytics = () => {
    const { shortID } = useParams();
    const [data, setData] = useState(null);
    const [error, setError] = useState('');

    useEffect(() => {
        const fetchData = async () => {
            try {
                const response = await fetch(`${BaseURL}/api/analytic/stats/${shortID}`);
                if (!response.ok) throw new Error('Failed to fetch data');
                const result = await response.text();
                setData(result);
            } catch (error) {
                setError(error.message);
            }
        };

        fetchData();
    }, [shortID]);

    return (
        <div className="container">
            <h1>Analytics</h1>
            {error && <p className="error">{error}</p>}
            {data ? (
                <div className="analytics-data">
                    <p>{data}</p> {/* Adjust this based on your backend response */}
                </div>
            ) : (
                <p>Loading...</p>
            )}
        </div>
    );
};

export default Analytics;
