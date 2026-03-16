package com.pesira.traceability.helpers;

import com.google.api.client.auth.oauth2.Credential;
import com.google.api.client.extensions.java6.auth.oauth2.AuthorizationCodeInstalledApp;
import com.google.api.client.extensions.jetty.auth.oauth2.LocalServerReceiver;
import com.google.api.client.googleapis.auth.oauth2.GoogleAuthorizationCodeFlow;
import com.google.api.client.googleapis.auth.oauth2.GoogleClientSecrets;
import com.google.api.client.googleapis.javanet.GoogleNetHttpTransport;
import com.google.api.client.http.javanet.NetHttpTransport;
import com.google.api.client.json.JsonFactory;
import com.google.api.client.json.gson.GsonFactory;
import com.google.api.services.gmail.GmailScopes;
import com.pesira.traceability.repository.UtilRepository;
import io.github.cdimascio.dotenv.Dotenv;
import java.io.FileInputStream;
import java.io.InputStreamReader;
import java.util.Collections;
import java.util.List;
import java.util.Optional;

public class GmailOauth {

    private static final Dotenv dotenv = Dotenv.configure().ignoreIfMissing().load();
    private static final JsonFactory JSON_FACTORY = GsonFactory.getDefaultInstance();
    private static final List<String> SCOPES = Collections.singletonList(GmailScopes.GMAIL_SEND);

    public static void main(String[] args) throws Exception {
        String credentialsFilePath = Optional.ofNullable(System.getenv("GOOGLE_OAUTH_CREDENTIALS")).orElse(
                dotenv.get("GOOGLE_OAUTH_CREDENTIALS"));
        if (credentialsFilePath == null || credentialsFilePath.isEmpty())
            throw new IllegalAccessException(
                    "ENV variable GOOGLE_OAUTH_CREDENTIALS not found");

        System.out.println(credentialsFilePath);
        UtilRepository utilRepository = new UtilRepository();

        NetHttpTransport httpTransport = GoogleNetHttpTransport.newTrustedTransport();
        InputStreamReader isr = new InputStreamReader(new FileInputStream(credentialsFilePath));
        System.out.println(isr);

        GoogleClientSecrets clientSecrets = GoogleClientSecrets.load(JSON_FACTORY, isr);

        GoogleAuthorizationCodeFlow flow = new GoogleAuthorizationCodeFlow.Builder(
                httpTransport,
                JSON_FACTORY,
                clientSecrets,
                SCOPES)
                .setAccessType("offline")
                .build();

        Credential credential = new AuthorizationCodeInstalledApp(flow, new LocalServerReceiver()).authorize("user");
        utilRepository.updateTokenEntities(credential.getAccessToken(), credential.getRefreshToken());
        System.out.println("Oauth tokens gotten");
    }
}
