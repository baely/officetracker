document.addEventListener('DOMContentLoaded', () => {
    // --- City Animation ---
    // The scrolling animation is primarily handled by CSS.
    // JavaScript might be used here if we need to adjust speed or change animation based on conditions.
    console.log("Infinite City theme script loaded.");

    // --- User Location & Weather ---
    const weatherWidget = document.getElementById('weather-widget');
    const temperatureDisplay = document.getElementById('temperature');
    const conditionDisplay = document.getElementById('condition');
    const locationDisplay = document.getElementById('location');

    function fetchWeather(latitude, longitude) {
        if (!weatherWidget || !temperatureDisplay || !conditionDisplay || !locationDisplay) {
            console.log("Weather display elements not found. Skipping weather fetch.");
            return;
        }

        const apiUrl = `https://api.open-meteo.com/v1/forecast?latitude=${latitude}&longitude=${longitude}&current_weather=true&hourly=temperature_2m,relativehumidity_2m,precipitation_probability,weathercode`;

        fetch(apiUrl)
            .then(response => response.json())
            .then(data => {
                if (data && data.current_weather) {
                    const weather = data.current_weather;
                    temperatureDisplay.textContent = `${weather.temperature}`;
                    conditionDisplay.textContent = getWeatherDescription(weather.weathercode);
                    // Location is already known, but you could use a reverse geocoding API here if desired
                    // For now, we'll just indicate that weather data is for the fetched location.
                    // locationDisplay.textContent = "Current Location"; // Or more specific if available

                    updateThemeBasedOnWeather(weather.weathercode, data.current_weather.is_day);
                } else {
                    conditionDisplay.textContent = "Weather data unavailable.";
                }
            })
            .catch(error => {
                console.error("Error fetching weather data:", error);
                if (conditionDisplay) conditionDisplay.textContent = "Could not fetch weather.";
            });
    }

    function getWeatherDescription(code) {
        // Based on WMO Weather interpretation codes (simplified)
        const descriptions = {
            0: "Clear sky",
            1: "Mainly clear", 2: "Partly cloudy", 3: "Overcast",
            45: "Fog", 48: "Depositing rime fog",
            51: "Light drizzle", 53: "Moderate drizzle", 55: "Dense drizzle",
            56: "Light freezing drizzle", 57: "Dense freezing drizzle",
            61: "Slight rain", 63: "Moderate rain", 65: "Heavy rain",
            66: "Light freezing rain", 67: "Heavy freezing rain",
            71: "Slight snow fall", 73: "Moderate snow fall", 75: "Heavy snow fall",
            77: "Snow grains",
            80: "Slight rain showers", 81: "Moderate rain showers", 82: "Violent rain showers",
            85: "Slight snow showers", 86: "Heavy snow showers",
            95: "Thunderstorm", // Slight or moderate
            96: "Thunderstorm with slight hail", 99: "Thunderstorm with heavy hail"
        };
        return descriptions[code] || "Unknown";
    }

    function getUserLocation() {
        if (!weatherWidget || !locationDisplay) {
            console.log("Weather or location display elements not found. Skipping location fetch.");
            return;
        }
        if (navigator.geolocation) {
            navigator.geolocation.getCurrentPosition(position => {
                const { latitude, longitude } = position.coords;
                locationDisplay.textContent = `Lat: ${latitude.toFixed(2)}, Lon: ${longitude.toFixed(2)}`;
                fetchWeather(latitude, longitude);
            }, error => {
                console.error("Error getting location:", error);
                locationDisplay.textContent = "Location access denied.";
                // Fallback: Try fetching weather for a default location or let user input
                // For now, just indicate the issue.
                if (conditionDisplay) conditionDisplay.textContent = "Enable location for weather.";
            });
        } else {
            locationDisplay.textContent = "Geolocation not supported.";
            if (conditionDisplay) conditionDisplay.textContent = "Location services needed.";
        }
    }

    // --- Theme Updates based on Time/Weather ---
    function updateThemeBasedOnWeather(weatherCode, isDay) {
        const body = document.body;
        const cityBackground = document.getElementById('city-background');

        // Adjust background or other elements based on weather
        // Example: Change background tint or add rain/snow effect overlays
        console.log(`Updating theme for weather code: ${weatherCode}, Is day: ${isDay}`);

        // Remove previous weather classes
        body.className = body.className.replace(/\bweather-\S+/g, '');


        if (!isDay) {
            body.classList.add('theme-night');
        } else {
            body.classList.remove('theme-night');
        }

        // Add classes for specific weather conditions
        if ([0, 1].includes(weatherCode)) { // Clear or mainly clear
            body.classList.add('weather-clear');
            if (cityBackground) cityBackground.style.filter = "brightness(100%)";
        } else if ([2, 3].includes(weatherCode)) { // Cloudy / Overcast
            body.classList.add('weather-cloudy');
            if (cityBackground) cityBackground.style.filter = "brightness(80%) sepia(10%)";
        } else if (String(weatherCode).startsWith('5') || String(weatherCode).startsWith('6') || String(weatherCode).startsWith('80')) { // Drizzle/Rain
            body.classList.add('weather-rain');
            if (cityBackground) cityBackground.style.filter = "brightness(70%) hue-rotate(10deg)";
            // Potentially add a rain effect overlay here
        } else if (String(weatherCode).startsWith('7') || String(weatherCode).startsWith('85') || String(weatherCode).startsWith('86')) { // Snow
            body.classList.add('weather-snow');
            if (cityBackground) cityBackground.style.filter = "brightness(90%) contrast(110%)";
            // Potentially add a snow effect overlay here
        } else if ([45, 48].includes(weatherCode)) { // Fog
            body.classList.add('weather-fog');
            if (cityBackground) cityBackground.style.filter = "brightness(75%) opacity(80%)";
        } else if (String(weatherCode).startsWith('9')) { // Thunderstorm
             body.classList.add('weather-thunderstorm');
             if (cityBackground) cityBackground.style.animationName = "thunderstormFlash, scrollCity"; // Example
        }

        // Adjust scrolling speed based on time or weather (optional)
        // Example: Slower scroll at night
        if (cityBackground) {
            if (!isDay) {
                cityBackground.style.animationDuration = "120s"; // Slower scroll at night
            } else {
                cityBackground.style.animationDuration = "60s"; // Default speed
            }
        }
    }
    
    // Initialize
    if (document.getElementById('weather-widget')) { // Only run if weather widget exists on the page
        getUserLocation();
    }

    // Example: Update theme every 30 minutes
    // setInterval(getUserLocation, 30 * 60 * 1000); 
});

// --- Calendar Specific Logic (if any, can be in a separate file or here) ---
// This will be for handling date clicks, month changes, etc.
// For now, it's assumed to be part of a different script or added later.
// Example:
// const days = document.querySelectorAll('.day');
// days.forEach(day => {
// day.addEventListener('click', () => {
// console.log('Day clicked:', day.textContent);
// // Add logic to handle day selection
// });
// });
