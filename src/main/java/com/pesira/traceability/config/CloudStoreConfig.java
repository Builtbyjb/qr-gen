package com.pesira.traceability.config;

import com.google.cloud.WriteChannel;
import com.google.cloud.storage.BlobId;
import com.google.cloud.storage.BlobInfo;
import com.google.cloud.storage.Storage;
import com.google.cloud.storage.StorageOptions;
import io.github.cdimascio.dotenv.Dotenv;
import java.io.IOException;
import java.net.URL;
import java.nio.ByteBuffer;
import java.nio.channels.FileChannel;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardOpenOption;
import java.util.Optional;
import java.util.concurrent.TimeUnit;

public class CloudStoreConfig {

    private static final Dotenv dotenv = Dotenv.configure().ignoreIfMissing().load();
    private final String BUCKET_NAME;
    private final Storage storage;

    public CloudStoreConfig() {
        BUCKET_NAME = Optional.ofNullable(System.getenv("BUCKET_NAME")).orElse(dotenv.get("BUCKET_NAME"));
        if (BUCKET_NAME == null || BUCKET_NAME.isEmpty()) {
            throw new IllegalArgumentException("ENV variable BUCKET_NAME not found");
        }

        this.storage = StorageOptions.getDefaultInstance().getService();
    }

    public String uploadFile(String fileName, String dir) throws IOException {
        Path filePath = Paths.get(dir + fileName);
        BlobId blobId = BlobId.of(BUCKET_NAME, fileName);
        BlobInfo blobInfo = BlobInfo.newBuilder(blobId).setContentType("application/zip").build();

        // Upload the file
        // storage.create(blobInfo, Files.readAllBytes(Paths.get(filePath)));
        System.out.println("Started streaming file...");
        try (
            WriteChannel writer = storage.writer(blobInfo);
            FileChannel fileChannel = FileChannel.open(filePath, StandardOpenOption.READ)
        ) {
            ByteBuffer buffer = ByteBuffer.allocate(1024 * 1024 * 1024); // 1MB buffer
            while (fileChannel.read(buffer) > 0) {
                buffer.flip();
                writer.write(buffer);
                buffer.clear();
            }
        }
        System.out.println("Finished streaming file!!");

        // Download link expires in 24 hours
        URL signedUrl = storage.signUrl(blobInfo, 24, TimeUnit.HOURS, Storage.SignUrlOption.withV4Signature());

        return signedUrl.toString();
    }
}
