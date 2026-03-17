package com.builtbyjb.qrgen.service;

import com.builtbyjb.qrgen.config.CloudStoreConfig;
import com.builtbyjb.qrgen.config.GmailConfig;
import com.builtbyjb.qrgen.helpers.Context;
import com.builtbyjb.qrgen.model.PartnerModel;
import com.builtbyjb.qrgen.repository.UtilRepository;
import jakarta.mail.MessagingException;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.GeneralSecurityException;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.zip.ZipEntry;
import java.util.zip.ZipOutputStream;

public class QRCodeService {

    private static final int CPU_CORES = Runtime.getRuntime().availableProcessors(); // Number of logical cpu cores
    private static final ExecutorService EXECUTOR = Executors.newFixedThreadPool(Math.min(CPU_CORES, 4));
    private static final UtilRepository dataStore = new UtilRepository();
    private static final CloudStoreConfig cloudStoreConfig = new CloudStoreConfig();
    private static final CodeGenService codeGenService = new CodeGenService();
    private static final String TMP_ZIP_DIR = "./tmp/zips/";
    private static final String TMP_PDF_DIR = "./tmp/pdfs/";
    private Long PROJECT_ID;

    public boolean generateQRCodes(
            String userId,
            String userEmail,
            Long projectId,
            String projectName,
            float size,
            long quantity,
            String location,
            List<String> logos) throws IOException, GeneralSecurityException, MessagingException {
        this.PROJECT_ID = projectId;

        long cursor;
        if (Context.DEBUG.equalsValue(0) && Context.TEST.equalsValue(0)) {
            cursor = dataStore.getCursorEntity();
        } else {
            cursor = 0;
        }

        List<PartnerModel> partners = new ArrayList<>();
        partners.add(new PartnerModel("SYDIP", 36));

        int totalQRCodeCount = 0;
        for (PartnerModel partner : partners) {
            totalQRCodeCount += partner.getHouseholds();
        }
        System.out.println("Total QR code count: " + totalQRCodeCount);

        long end = cursor + totalQRCodeCount;

        // Store generated codes
        List<String> generatedCodes = new ArrayList<>();
        // Generate QR codes
        for (long i = cursor; i < end; i++) {
            String qrCode = codeGenService.generateQRCode(i);
            generatedCodes.add(qrCode);
        }

        if (generatedCodes.size() != totalQRCodeCount) {
            throw new IllegalStateException("Generated QR codes count does not match expected count");
        }

        if (!validateQRCodes(generatedCodes)) {
            throw new IllegalStateException("Duplicate QR codes generated");
        }

        // Generate PDFs
        int count = 0;
        int chunkSize = 500;

        for (PartnerModel partner : partners) {
            int partnerQRCodesCount = partner.getHouseholds();
            showProgress(count, totalQRCodeCount);
            List<String> trimmedCodes = trimList(generatedCodes, partnerQRCodesCount);
            if (trimmedCodes.size() != partnerQRCodesCount) {
                throw new IllegalStateException("Trimmed QR codes count does not match expected count");
            }
            generatePDFs(chunkSize, trimmedCodes, projectName, logos, size, partner.getName(), location);
            count += partnerQRCodesCount;
            showProgress(count, totalQRCodeCount);
        }

        shutdownExecutor();

        // Update Datastore cursor
        if (Context.DEBUG.equalsValue(0)) {
            dataStore.updateCursorEntity(end);
        }

        // Zip PDFs
        List<String> folderNames = getFolderPaths(TMP_PDF_DIR);
        String zipFileName = zipPDFs(folderNames, projectName);
        if (Context.DEBUG.greaterThanOrEqual(1)) {
            System.out.println("Generated zip file: " + zipFileName);
        }

        // Upload to cloud storage
        String fileLink;
        if (Context.DEBUG.equalsValue(0)) {
            fileLink = cloudStoreConfig.uploadFile(zipFileName, TMP_ZIP_DIR);
        } else {
            fileLink = "https://demo_file_link.zip";
        }

        // Email link to user
        GmailConfig gmailConfig = new GmailConfig();
        String subject = "QR Codes for " + projectName;
        String mail = String.format(
                """
                        Hello, your QR codes for %s are ready.
                        <br/>
                        <br/>
                        Click <a href=\"%s\">here</a> to download them.
                        <br/>
                        <br/>
                        Thank you for using Pesira.
                        <br/>
                        <br/>
                        """,
                projectName,
                fileLink);

        if (Context.DEBUG.equalsValue(0)) {
            gmailConfig.sendEmail(userEmail, subject, mail);
        }

        // Clean up
        cleanUp(logos, zipFileName, folderNames);

        return true;
    }

    private void generatePDFs(
            int chunkSize,
            List<String> generatedCodes,
            String projectName,
            List<String> logos,
            float size,
            String partner,
            String location) {
        List<CompletableFuture<Void>> futures;
        // Create directory if not exists
        createDirectory(TMP_PDF_DIR);

        // Create folder if not exits
        String folderName = TMP_PDF_DIR + partner + "_" + String.valueOf(System.currentTimeMillis());
        createDirectory(folderName);

        List<List<String>> qrCodeChunks = chunkList(generatedCodes, chunkSize);
        AtomicInteger index = new AtomicInteger(0);

        futures = qrCodeChunks
                .stream()
                .map(qrCodeChunk -> {
                    int idx = index.getAndIncrement();
                    return CompletableFuture.runAsync(
                            () -> {
                                try {
                                    PDFGenService.generatePDF(
                                            idx,
                                            projectName,
                                            PROJECT_ID,
                                            qrCodeChunk,
                                            logos,
                                            size,
                                            partner,
                                            location,
                                            folderName);
                                } catch (Exception e) {
                                    e.printStackTrace();
                                }
                            },
                            EXECUTOR);
                })
                .toList();

        CompletableFuture.allOf(futures.toArray(new CompletableFuture[0])).join();
    }

    // Splits a large list into a list of smaller lists
    private List<List<String>> chunkList(List<String> list, int chunkSize) {
        List<List<String>> chunks = new ArrayList<>();
        for (int i = 0; i < list.size(); i += chunkSize) {
            chunks.add(list.subList(i, Math.min(i + chunkSize, list.size())));
        }
        return chunks;
    }

    private List<String> trimList(List<String> generatedCodes, int count) {
        List<String> trimmedList = new ArrayList<>();
        for (int i = 0; i < count && !generatedCodes.isEmpty(); i++) {
            trimmedList.add(generatedCodes.remove(0));
        }
        return trimmedList;
    }

    public boolean validateQRCodes(List<String> qrCodes) {
        Set<String> codes = new HashSet<>();
        for (String qr : qrCodes) {
            if (!codes.add(qr)) {
                System.err.println("Duplicate QR code found: " + qr);
                return false;
            }
        }
        return true;
    }

    public void showProgress(int current, int total) {
        float percentage = (float) (current * 100) / (float) total;
        String formatted = String.format("%.2f", percentage);
        System.out.print("\rProgress: " + formatted + "%");
        System.out.flush();
    }

    private String zipPDFs(List<String> folderNames, String projectName) {
        // Create directory if not exists
        createDirectory(TMP_ZIP_DIR);
        String idx = String.valueOf(System.currentTimeMillis());
        String zipFileName = "qr_codes_" + projectName + "_" + idx + ".zip";

        try (ZipOutputStream zipOut = new ZipOutputStream(new FileOutputStream(TMP_ZIP_DIR + zipFileName))) {
            for (String folderName : folderNames) {
                Path folderPath = Paths.get(folderName);

                // Recursively walk the folder
                Files.walk(folderPath)
                        .filter(Files::isRegularFile)
                        .forEach(path -> {
                            try (FileInputStream pdfIn = new FileInputStream(path.toFile())) {
                                // preserve folder structure inside the zip
                                String zipEntryName = folderPath.getFileName() + "/" + folderPath.relativize(path);
                                ZipEntry zipEntry = new ZipEntry(zipEntryName);
                                zipOut.putNextEntry(zipEntry);

                                byte[] buffer = new byte[8192];
                                int length;
                                while ((length = pdfIn.read(buffer)) >= 0) {
                                    zipOut.write(buffer, 0, length);
                                }

                                zipOut.closeEntry();
                            } catch (IOException e) {
                                throw new RuntimeException("Error creating zip file: " + e.getMessage());
                            }
                        });
            }
        } catch (IOException e) {
            throw new RuntimeException("Error creating zip folder: " + e.getMessage());
        }
        return zipFileName;
    }

    private void createDirectory(String dirPath) {
        try {
            Path path = Paths.get(dirPath);
            if (!Files.exists(path)) {
                Files.createDirectories(path);
            }
        } catch (IOException e) {
            System.err.println("Error creating directory: " + e.getMessage());
        }

        if (Context.DEBUG.greaterThanOrEqual(2))
            System.out.println("Directory created");
    }

    private void cleanUp(List<String> logoPaths, String zipFileName, List<String> folderNames) {
        if (Context.DEBUG.greaterThanOrEqual(1)) {
            System.out.println("Starting cleanup...");
        }

        // Delete image files in the temp directory
        // for (String logoPath : logoPaths) {
        //     try {
        //         Files.deleteIfExists(Paths.get(logoPath));
        //     } catch (IOException e) {
        //         System.err.println("Error deleting logo file: " + e.getMessage());
        //     }
        // }

        // Delete zip file
        if (Context.DEBUG.equalsValue(0)) {
            try {
                Files.deleteIfExists(Paths.get(TMP_ZIP_DIR + zipFileName));
            } catch (IOException e) {
                System.err.println("Error deleting zip file: " + e.getMessage());
            }
        }

        // Delete PDF files
        // if (Context.DEBUG.equalsValue(0)) {
        //     for (String folderName : folderNames) {
        //         try {
        //             Files.walk(Paths.get(folderName))
        //                     .sorted(Comparator.reverseOrder())
        //                     .forEach(path -> {
        //                         try {
        //                             Files.delete(path);
        //                         } catch (IOException e) {
        //                             e.printStackTrace();
        //                         }
        //                     });
        //         } catch (IOException e) {
        //             System.err.println("Error deleting PDF file: " + e.getMessage());
        //         }
        //     }
        // }

        if (Context.DEBUG.greaterThanOrEqual(1)) {
            System.out.println("cleanup completed!!");
        }
    }

    public static void shutdownExecutor() {
        EXECUTOR.shutdown();
        try {
            if (!EXECUTOR.awaitTermination(30, TimeUnit.SECONDS)) {
                EXECUTOR.shutdownNow();
            }
        } catch (InterruptedException e) {
            EXECUTOR.shutdownNow();
            Thread.currentThread().interrupt();
        }
    }

    public List<String> getFolderPaths(String directoryPath) {
        File directory = new File(directoryPath);
        List<String> folderPaths = new ArrayList<>();
        if (directory.exists() && directory.isDirectory()) {
            File[] files = directory.listFiles();
            if (files != null) {
                for (File file : files) {
                    if (file.isDirectory()) {
                        folderPaths.add(file.getAbsolutePath());
                    }
                }
            }
        }
        return folderPaths;
    }
}
