'use client';

import React, { useState, useEffect, useCallback } from 'react';
import Calendar from 'react-calendar';
import 'react-calendar/dist/Calendar.css';
import { format } from 'date-fns';
import axios from 'axios';

type WeatherEntry = {
  date: string;
  location: string;
  temperature: number;
  condition: string;
};

export default function WeatherCalendar() {
  const ONE_DAY_IN_MILLI = 24 * 60 * 60 * 1000; // milliseconds in one day
  
  const [selectedDate, setSelectedDate] = useState<Date>(new Date(new Date().getTime() - (ONE_DAY_IN_MILLI * 3))); // Default to 2 days ago
  const [weatherData, setWeatherData] = useState<WeatherEntry | null>({} as WeatherEntry);
  const [chicagoData, setChicagoData] = useState<WeatherEntry | null>({} as WeatherEntry);
  const [location, setSelectedLocation] = useState<string>('Akron'); // Default location
  const [loading, setLoading] = useState<boolean>(true);
  const [chicagoLoading, setChicagoLoading] = useState<boolean>(true);
  const [isMounted, setIsMounted] = useState<boolean>(false);

  useEffect(() => {
    setIsMounted(true);
  }, []);

  const url = 'http://192.168.0.10:8080'

  useEffect(() => {
    const fetchChicagoWeather = async () => {
      const dateStr = format(selectedDate, 'yyyy-MM-dd');
      try {
        const res = await axios.get(`${url}/weather?location=chicago&date=${dateStr}`);
        console.log('Weather data:', res.data);
        setChicagoData(res.data);
      } catch (err) {
        // console.error('Failed to fetch weather', err);
        setChicagoData(null);
      }
    };
    const timer = setTimeout(() => {
      fetchChicagoWeather().then(() => setChicagoLoading(false));
    }, 500);

    return () => clearTimeout(timer);
  }, [selectedDate]);

  useEffect(() => {
    const fetchWeather = async () => {
      const dateStr = format(selectedDate, 'yyyy-MM-dd');
      try {
        const res = await axios.get(`${url}/weather?location=${location}&date=${dateStr}`);
        console.log('Weather data:', res.data);
        setWeatherData(res.data);
      } catch (err) {
        // console.error('Failed to fetch weather', err);
        setWeatherData(null);
      }
    };
    const timer = setTimeout(() => {
      fetchWeather().then(() => setLoading(false));
    }, 500);

    return () => clearTimeout(timer);
  }, [selectedDate, location]);

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <h1 className="text-3xl font-bold mb-4 text-center">Average Weather vs Chicago by Date</h1>
        <h2 className="text-xl mb-4 text-center">ily baby!</h2>
      <div className="mb-6">
        <Calendar
          onChange={(value) => {setSelectedDate(value as Date); setLoading(true); setChicagoLoading(true);}}
          value={selectedDate}
          className="mx-auto max-w-md"
          tileDisabled={({ date }) => {
            return date.getTime() >= new Date().getTime() - (ONE_DAY_IN_MILLI * 3);
          } // Disable future dates and today minus 2 days
        }
        />
      </div>
        {isMounted ? (
    <div className="mt-6 text-center">
        <input type="text" className="border rounded p-2 w-full max-w-md mx-auto mb-4"
         value={location} placeholder="Enter location cutie" 
        onChange={(value) => {
          setSelectedLocation(value.target.value);
          setLoading(true);
      }} />
      </div>
        ) : null
      }

      {weatherData ? (
        <div className="max-w-xl mx-auto rounded-xl shadow-md p-6 text-center">
          <h2 className="text-xl font-semibold mb-2">{format(selectedDate, 'PPP')}</h2>
          <p className="text-lg mb-2">{weatherData.location}</p>
          {loading ? (<p className="text-2xl font-bold mb-2">---</p>) : (<p className="text-2xl font-bold mb-2">{weatherData.temperature.toFixed(1)}°F</p>)}
          <p className="text-md text-gray-600">{weatherData.condition}</p>
        </div>
      ) : null
    }

      {chicagoData ? (
        <div className="max-w-xl mx-auto rounded-xl shadow-md p-6 text-center">
          <h2 className="text-xl font-semibold mb-2">{format(selectedDate, 'PPP')}</h2>
          <p className="text-lg mb-2">Chicago, IL, United States</p>
          {chicagoLoading ? (<p className="text-2xl font-bold mb-2">---</p>) : (<p className="text-2xl font-bold mb-2">{chicagoData.temperature.toFixed(1)}°F</p>)}
          <p className="text-md text-gray-600">{chicagoData.condition}</p>
        </div>
      ) : null 
      }

    </div>
  );
}
