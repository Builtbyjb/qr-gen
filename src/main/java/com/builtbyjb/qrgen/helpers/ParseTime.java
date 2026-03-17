package com.builtbyjb.qrgen.helpers;

public class ParseTime {

    public String parseTime(double duration) {
        // For microseconds
        if (duration < 1) {
            return String.format("%.2f μs", duration * 1_000);
        }

        // For milliseconds
        if (duration < 1_000) {
            return String.format("%.2f ms", duration);
        }

        // For seconds
        double seconds = duration / 1_000.0;
        if (seconds < 60) {
            return String.format("%.2f s", seconds);
        }

        // For minutes
        double minutes = seconds / 60.0;
        if (minutes < 60) {
            int minutesInt = (int) minutes;
            int secondsInt = (int) (minutes - minutesInt) * 60;
            return String.format("%d min %d s", minutesInt, secondsInt);
        }

        // For hours
        double hours = minutes / 60.0;
        if (hours < 24) {
            int hoursInt = (int) hours;
            int minutesInt = (int) (hours - hoursInt) * 60;
            return String.format("%d h %d min", hoursInt, minutesInt);
        }

        return String.format("%.2f days", hours / 24);
    }
}
