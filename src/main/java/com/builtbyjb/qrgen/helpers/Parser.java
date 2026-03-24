package com.builtbyjb.qrgen.helpers;

import java.util.Optional;

import com.builtbyjb.qrgen.helpers.types.Argument;
import com.builtbyjb.qrgen.helpers.types.Storage;
import com.builtbyjb.qrgen.helpers.types.Format;

public class Parser {

    public static String parseTime(double duration) {
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

    public static Optional<Argument> parseArguments(String[] args, String version) {

        int quantity = 0; // Required
        String info = "";
        int width = 500;
        int height = 500;
        String url = ""; // Required
        Format format = Format.PDF; // Required;
        Storage storage = Storage.fromString("local");

        for (String arg : args) {
            String[] str = arg.split("=");
            switch (str[0]) {
                case "--version", "-v":
                    System.out.println("QR gen version " + version);
                    return null;
                case "--help", "-h":
                    System.out.println("Help/Usage");
                    return null;
                case "--quantity":
                    quantity = Integer.parseInt(str[1]);
                    break;
                case "--size":
                    String[] szs = str[1].split("x");
                    width = Integer.parseInt(szs[0]);
                    height = Integer.parseInt(szs[1]);
                    break;
                case "--info":
                    info = str[1];
                    break;
                case "--url":
                    url = str[1];
                    break;
                case "--format":
                    format = Format.fromString(str[1]);
                    break;
                case "--storage":
                    storage = Storage.fromString(str[1]);
                    break;
            }
        }

        if (quantity < 1) {
            System.out.println("Error: Quantity must be greater than 0");
            // Print usage instructions
            return null;
        }

        if (url.isEmpty()) {
            System.out.println("Error: URL is required");
            // Print usage instructions
            return null;
        }

        if (format != Format.PDF) {
            System.out.println("Error: Unsupported format");
            // Print usage instructions
            return null;
        }

        Argument argument = Argument.builder()
                .quantity(quantity)
                .info(info)
                .width(width)
                .height(height)
                .url(url)
                .format(format)
                .storage(storage)
                .build();

        return Optional.ofNullable(argument);
    }
}
