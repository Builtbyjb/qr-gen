package com.pesira.traceability.config;

import com.google.api.client.googleapis.javanet.GoogleNetHttpTransport;
import com.google.api.client.http.GenericUrl;
import com.google.api.client.http.HttpRequest;
import com.google.api.client.http.HttpRequestFactory;
import com.google.api.client.http.HttpRequestInitializer;
import com.google.api.client.http.HttpResponse;
import com.google.api.client.http.HttpTransport;
import com.google.api.client.http.UrlEncodedContent;
import com.google.api.client.json.JsonFactory;
import com.google.api.client.json.gson.GsonFactory;
import com.google.api.client.util.GenericData;
import com.google.api.services.gmail.Gmail;
import com.google.api.services.gmail.model.Message;
import com.pesira.traceability.repository.UtilRepository;
import io.github.cdimascio.dotenv.Dotenv;
import jakarta.mail.MessagingException;
import jakarta.mail.Session;
import jakarta.mail.internet.InternetAddress;
import jakarta.mail.internet.MimeMessage;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.security.GeneralSecurityException;
import java.util.HashMap;
import java.util.Map;
import java.util.Optional;
import java.util.Properties;
import org.apache.commons.codec.binary.Base64;

public class GmailConfig {

    private static final JsonFactory JSON_FACTORY = GsonFactory.getDefaultInstance();
    private static final Dotenv dotenv = Dotenv.configure().ignoreIfMissing().load();
    private final HttpTransport httpTransport;
    private final String clientId;
    private final String clientSecret;
    private final String senderEmail;
    private static final UtilRepository utilRepository = new UtilRepository();

    public GmailConfig() throws IOException, GeneralSecurityException {
        clientId = Optional.ofNullable(System.getenv("GOOGLE_CLIENT_ID")).orElse(dotenv.get("GOOGLE_CLIENT_ID"));
        if (clientId == null || clientId.isEmpty()) throw new IllegalArgumentException(
            "ENV variable CLIENT_ID not found"
        );

        clientSecret = Optional.ofNullable(System.getenv("GOOGLE_CLIENT_SECRET")).orElse(
            dotenv.get("GOOGLE_CLIENT_SECRET")
        );

        if (clientSecret == null || clientSecret.isEmpty()) throw new IllegalArgumentException(
            "ENV variable GOOGLE_CLIENT_SECRET not found"
        );

        senderEmail = Optional.ofNullable(System.getenv("SENDER_EMAIL")).orElse(dotenv.get("SENDER_EMAIL"));
        if (senderEmail == null || senderEmail.isEmpty()) throw new IllegalArgumentException(
            "ENV variable SENDER_EMAIL not set not found"
        );

        httpTransport = GoogleNetHttpTransport.newTrustedTransport();
    }

    public void sendEmail(String userEmail, String subject, String mail)
        throws IOException, GeneralSecurityException, MessagingException {
        Map<String, String> tokens = utilRepository.getTokenEntities();
        String accessToken = verifyAccessToken(tokens.get("accessToken"), tokens.get("refreshToken"));

        HttpRequestInitializer requestInitializer = request -> {
            request.getHeaders().setAuthorization("Bearer " + accessToken);
        };

        HttpTransport httpTransport = GoogleNetHttpTransport.newTrustedTransport();
        Gmail service = new Gmail.Builder(httpTransport, GsonFactory.getDefaultInstance(), requestInitializer)
            .setApplicationName("example")
            .build();

        // Encode as MIME message
        Properties props = new Properties();
        Session session = Session.getDefaultInstance(props, null);
        MimeMessage email = new MimeMessage(session);
        email.setFrom(new InternetAddress(senderEmail, "Pesira"));
        email.addRecipient(jakarta.mail.Message.RecipientType.TO, new InternetAddress(userEmail));
        email.setSubject(subject);
        email.setContent(mail, "text/html; charset=utf-8");

        // Encode and wrap the MIME message into a gmail message
        ByteArrayOutputStream buffer = new ByteArrayOutputStream();
        email.writeTo(buffer);
        byte[] rawMessageBytes = buffer.toByteArray();
        String encodedEmail = Base64.encodeBase64URLSafeString(rawMessageBytes);
        Message message = new Message();
        message.setRaw(encodedEmail);

        // Create send message
        service.users().messages().send("me", message).execute();
        System.out.println("Message id: " + message.getId());
        // System.out.println(message.toPrettyString());
        httpTransport.shutdown();
    }

    public String verifyAccessToken(String accessToken, String refreshToken) throws IOException {
        GenericUrl url = new GenericUrl("https://www.googleapis.com/oauth2/v1/tokeninfo?access_token=" + accessToken);
        HttpRequestFactory requestFactory = httpTransport.createRequestFactory();
        try {
            HttpRequest request = requestFactory.buildGetRequest(url);
            HttpResponse response = request.execute(); // If invalid, will throw IOException
            if (response.getStatusCode() == 200) {
                return accessToken;
            }
        } catch (IOException e) {
            return refreshAccessToken(refreshToken);
        }

        return null;
    }

    private String refreshAccessToken(String refreshToken) throws IOException {
        GenericUrl tokenUrl = new GenericUrl("https://oauth2.googleapis.com/token");
        HttpRequestFactory requestFactory = httpTransport.createRequestFactory();
        Map<String, String> params = new HashMap<>();
        params.put("client_id", clientId);
        params.put("client_secret", clientSecret);
        params.put("refresh_token", refreshToken);
        params.put("grant_type", "refresh_token");

        UrlEncodedContent content = new UrlEncodedContent(params);
        HttpRequest request = requestFactory.buildPostRequest(tokenUrl, content);
        HttpResponse response = request.execute();
        System.out.println(response.getStatusCode());
        GenericData jsonResponse = JSON_FACTORY.fromInputStream(response.getContent(), GenericData.class);
        String accessToken = (String) jsonResponse.get("access_token");

        if (accessToken == null) {
            throw new IOException("Failed to refresh access token");
        }

        utilRepository.updateTokenEntities(accessToken, refreshToken);
        return accessToken;
    }
}
