package com.builtbyjb.qrgen.service;

import com.builtbyjb.qrgen.config.CloudStoreConfig;
import com.builtbyjb.qrgen.config.GmailConfig;
import com.builtbyjb.qrgen.helpers.Context;
import com.builtbyjb.qrgen.helpers.types.Argument;
import com.builtbyjb.qrgen.helpers.types.Format;
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
    // private static final UtilRepository dataStore = new UtilRepository();
    // private static final CloudStoreConfig cloudStoreConfig = new
    // CloudStoreConfig();
    private static final CodeGenService codeGenService = new CodeGenService();
    private static final String TMP_ZIP_DIR = "./tmp/zips/";
    private static final String TMP_PDF_DIR = "./tmp/pdfs/";

    public boolean generateQRCodes(Argument args) throws IOException, GeneralSecurityException, MessagingException {

        long cursor = 0L;
        long end = args.getQuantity();

        // Store generated codes
        List<String> qrCodes = new ArrayList<>();
        // Generate QR codes
        for (long i = cursor; i < end; i++) {
            String qrCode = codeGenService.generateQRCode(i);
            qrCodes.add(qrCode);
        }

        long totalQRCodeCount = qrCodes.size();

        if (totalQRCodeCount != end) {
            throw new IllegalStateException("Generated QR codes quantity does not match expected quantity");
        }

        if (!validateQRCodes(qrCodes)) {
            throw new IllegalStateException("Duplicate QR codes generated");
        }

        // Generate PDFs
        int count = 0;
        int chunkSize = 500;

        if (args.getFormat() == Format.PDF) {
            generatePDFs(qrCodes, chunkSize, args);
            shutdownExecutor();

            // Zip PDFs
            List<String> folderNames = getFolderPaths(TMP_PDF_DIR);
            String zipFileName = zipPDFs(folderNames);
            if (Context.DEBUG.greaterThanOrEqual(1)) {
                System.out.println("Generated zip file: " + zipFileName);
            }

            // Create CSV file
            generateCSV(qrCodes);

            // Upload to cloud storage
            // String fileLink;
            // if (Context.DEBUG.equalsValue(0)) {
            // fileLink = cloudStoreConfig.uploadFile(zipFileName, TMP_ZIP_DIR);
            // } else {
            // fileLink = "https://demo_file_link.zip";
            // }

            // Clean up
            cleanUp(zipFileName, folderNames);
        }

        return true;
    }

    private void generatePDFs(List<String> qrCodes, int chunkSize, Argument args) {
        List<CompletableFuture<Void>> futures;
        // Create directory if not exists
        createDirectory(TMP_PDF_DIR);

        // Create folder if not exits
        String folderName = TMP_PDF_DIR + "_" + String.valueOf(System.currentTimeMillis());
        createDirectory(folderName);

        List<List<String>> qrCodeChunks = chunkList(qrCodes, chunkSize);
        AtomicInteger index = new AtomicInteger(0);

        futures = qrCodeChunks
                .stream()
                .map(qrCodeChunk -> {
                    int idx = index.getAndIncrement();
                    return CompletableFuture.runAsync(
                            () -> {
                                try {
                                    PDFGenService.generatePDF(idx, qrCodeChunk, folderName, args);
                                } catch (Exception e) {
                                    e.printStackTrace();
                                }
                            },
                            EXECUTOR);
                })
                .toList();

        CompletableFuture.allOf(futures.toArray(new CompletableFuture[0])).join();
    }

    private void generateCSV(List<String> qrCodes) throws IOException {
        String fileName = "qr_codes_" + "_" + System.currentTimeMillis() + ".csv";
        Path csvPath = Paths.get(TMP_PDF_DIR, fileName);
        try (FileOutputStream fos = new FileOutputStream(csvPath.toFile())) {
            for (String qrCode : qrCodes) {
                fos.write((qrCode + "\n").getBytes());
            }
        }
    }

    // Splits a large list into a list of smaller lists
    private List<List<String>> chunkList(List<String> list, int chunkSize) {
        List<List<String>> chunks = new ArrayList<>();
        for (int i = 0; i < list.size(); i += chunkSize) {
            chunks.add(list.subList(i, Math.min(i + chunkSize, list.size())));
        }
        return chunks;
    }

    /*
     * Returns a chunk of the specified size from the list and removes it from the
     * original list
     */
    private List<String> trimList(List<String> generatedCodes, int count) {
        List<String> trimmedList = new ArrayList<>();
        for (int i = 0; i < count && !generatedCodes.isEmpty(); i++) {
            trimmedList.add(generatedCodes.remove(0));
        }
        return trimmedList;
    }

    private boolean validateQRCodes(List<String> qrCodes) {
        Set<String> codes = new HashSet<>();
        for (String qr : qrCodes) {
            if (!codes.add(qr)) {
                System.err.println("Duplicate QR code found: " + qr);
                return false;
            }
        }
        return true;
    }

    private void showProgress(int current, Long total) {
        float percentage = (float) (current * 100) / (float) total;
        String formatted = String.format("%.2f", percentage);
        System.out.print("\rProgress: " + formatted + "%");
        System.out.flush();
    }

    private String zipPDFs(List<String> folderNames) {
        // Create directory if not exists
        createDirectory(TMP_ZIP_DIR);
        String idx = String.valueOf(System.currentTimeMillis());
        String zipFileName = "qr_codes_" + "_" + idx + ".zip";

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

    private void cleanUp(String zipFileName, List<String> folderNames) {
        if (Context.DEBUG.greaterThanOrEqual(1)) {
            System.out.println("Starting cleanup...");
        }

        // Delete image files in the temp directory
        // for (String logoPath : logoPaths) {
        // try {
        // Files.deleteIfExists(Paths.get(logoPath));
        // } catch (IOException e) {
        // System.err.println("Error deleting logo file: " + e.getMessage());
        // }
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
        // for (String folderName : folderNames) {
        // try {
        // Files.walk(Paths.get(folderName))
        // .sorted(Comparator.reverseOrder())
        // .forEach(path -> {
        // try {
        // Files.delete(path);
        // } catch (IOException e) {
        // e.printStackTrace();
        // }
        // });
        // } catch (IOException e) {
        // System.err.println("Error deleting PDF file: " + e.getMessage());
        // }
        // }
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
