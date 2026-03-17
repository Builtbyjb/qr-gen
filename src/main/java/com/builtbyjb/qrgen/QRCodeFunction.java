package com.builtbyjb.qrgen;

import com.google.cloud.functions.HttpFunction;
import com.google.cloud.functions.HttpRequest;
import com.google.cloud.functions.HttpResponse;
import com.builtbyjb.qrgen.config.PostgresConfig;
import com.builtbyjb.qrgen.helpers.Context;
import com.builtbyjb.qrgen.helpers.ParseTime;
import com.builtbyjb.qrgen.service.QRCodeService;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardCopyOption;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class QRCodeFunction implements HttpFunction {

    private final QRCodeService qrCodeService;
    private final PostgresConfig dbConfig;

    public QRCodeFunction() {
        this.qrCodeService = new QRCodeService();
        this.dbConfig = new PostgresConfig();
        dbConfig.init();
        if (Context.DEBUG.greaterThanOrEqual(1)) System.out.println("Database tables initialized");
    }

    @Override
    public void service(HttpRequest request, HttpResponse response) throws IOException {
        response.setContentType("application/json");
        try {
            if (!"POST".equals(request.getMethod())) {
                response.setStatusCode(400);
                response.getWriter().write("Invalid request method, Try again with a POST request");
                return;
            }

            // Get Logos
            // TODO: Verify logo format
            List<String> logos = new ArrayList<>();
            String tempDir = System.getProperty("java.io.tmpdir");

            try {
                for (HttpRequest.HttpPart httpPart : request.getParts().values()) {
                    if (httpPart.getFileName().isEmpty()) {
                        continue;
                    }

                    String filename = httpPart.getFileName().get();
                    if (filename == null) {
                        continue;
                    }

                    Path filePath = Paths.get(tempDir, filename).toAbsolutePath();
                    Files.copy(httpPart.getInputStream(), filePath, StandardCopyOption.REPLACE_EXISTING);
                    logos.add(filePath.toString());
                }
            } catch (IllegalStateException e) {
                response.setStatusCode(400);
                response.getWriter().write("Duplicate Keys");
                return;
            }

            // Get form values
            Map<String, String> formFields = new HashMap<>();
            request.getQueryParameters()
                    .forEach((name, values) -> {
                        if (!values.isEmpty()) {
                            formFields.put(name, values.get(0));
                        }
                    });

            String userId = formFields.get("user_id");
            String userEmail = formFields.get("user_email");
            Long projectId = Long.valueOf(formFields.get("project_id"));
            String projectName = formFields.get("project_name");
            long quantity = Long.parseLong(formFields.get("quantity"));
            String location = formFields.get("location");

            // Input validation
            if (userId == null || userId.isEmpty()) {
                response.setStatusCode(400);
                response.getWriter().write("User id is required");
                return;
            }

            // TODO: Verify email format
            if (userEmail == null || userEmail.isEmpty()) {
                response.setStatusCode(400);
                response.getWriter().write("User email is required");
                return;
            }

            if (projectId == null) {
                response.setStatusCode(400);
                response.getWriter().write("Product id is required");
                return;
            }

            if (projectName == null || projectName.isEmpty()) {
                response.setStatusCode(400);
                response.getWriter().write("Project name is required");
                return;
            }

            if (location == null || location.isEmpty()) {
                response.setStatusCode(400);
                response.getWriter().write("Location is required");
                return;
            }

            String size = formFields.get("size");
            if (size == null || size.isEmpty()) size = "500";
            float sizeFloat = Float.parseFloat(size);

            long maxQRCodes = 500_000;
            if (quantity < 1 || quantity > maxQRCodes) {
                response.setStatusCode(400);
                response.getWriter().write("Invalid quantity amount. Quantity must be between 1 and 500,000.");
                return;
            }

            if (logos.size() > 4) {
                response.setStatusCode(400);
                response.getWriter().write("Invalid number of logos. The maximum number of logos allowed is 4.");
                return;
            }

            if (Context.DEBUG.greaterThanOrEqual(2)) {
                System.out.println("FormField UserId: " + userId);
                System.out.println("FormField UserEmail: " + userEmail);
                System.out.println("FormField ProductId: " + projectId);
                System.out.println("FormField Size: " + sizeFloat);
                System.out.println("FormField ProjectName: " + projectName);
                System.out.println("FormField Quantity: " + quantity);
                System.out.println("FormField Logos: " + logos.size());
            }

            double startTime = System.currentTimeMillis();
            try {
                boolean result = qrCodeService.generateQRCodes(
                        userId,
                        userEmail,
                        projectId,
                        projectName,
                        sizeFloat,
                        quantity,
                        location,
                        logos);
                if (result) {
                    response.setStatusCode(200);
                    response.getWriter().write("QR codes generated successfully");
                    return;
                }
            } catch (Exception e) {
                e.printStackTrace();
            } finally {
                ParseTime parseTime = new ParseTime();
                double endTime = System.currentTimeMillis();
                double duration = endTime - startTime;
                String timeTaken = parseTime.parseTime(duration);
                if (Context.DEBUG.greaterThanOrEqual(1)) System.out.println("Time taken: " + timeTaken);
            }

            response.setStatusCode(500);
            response.getWriter().write("Failed to generate QR codes");
        } catch (IOException e) {
            System.out.println("Error: " + e.getMessage());
            response.setStatusCode(500);
            response.getWriter().write("Failed to generate QR codes");
        }
    }
}
