# PESIRA Product Traceability System

A Google Cloud Function-based service for generating QR codes in bulk for product traceability. The system generates unique QR codes, embeds them in PDFs with custom logos, and provides download links via email.

## Overview

The traceability system generates unique QR codes that link to `https://www.pesira.io/traceability/{qr-code}` where each QR code follows the format `QR-XXXXXXXXXXXX` (12-character Base62 encoded identifier).

> [!WARNING]
> Testing and code run requires a unix environment (mac os or linux).

## API Documentation

```
POST /
Content-Type: multipart/form-data
```

### Request Parameters

| Parameter | Type | Required | Constraints |
|-----------|------|----------|-------------|
| `user_id` | string | ✅ | - |
| `project_id` | string | ✅ | - |
| `project_name` | string | ✅ | - |
| `location` | string | ✅ | - |
| `supplier_code` | string | ✅ | - |
| `quantity` | number | ✅ | 1 - 500,000 |
| `size` | number | ❌ | Default: 500px |
| `logo1` | Image | ❌ | formats: jpg, jpeg, png |
| `logo2` | Image | ❌ | formats: jpg, jpeg, png |
| `logo3` | Image | ❌ | formats: jpg, jpeg, png |
| `logo4` | Image | ❌ | formats: jpg, jpeg, png |

### Request Example

#### Using cURL

```bash
curl --request POST \
  --header "Content-Type: multipart/form-data" \
  --form "user_id=user123" \
  --form "project_id=prod456" \
  --form "project_name=project" \
  --form "location=SK" \
  --form "supplier_code=001" \
  --form "quantity=35" \
  --form "size=500" \
  --form "logo0=@/path/to/logo1.png" \
  --form "logo1=@/path/to/logo2.png" \
  --form "logo2=@/path/to/logo3.png" \
  --form "logo3=@/path/to/logo4.png" \
  https://CLOUD_FUNCTION_URL
```

### Response Format

#### Success Response (200)

```
Content-Type: application/json
Status: 200 OK

"QR codes generated successfully"
```

#### Error Responses

| Status Code | Description | Response Body |
|-------------|-------------|---------------|
| 400 | Bad Request | `"Invalid request method, Try again with a POST request"` |
| 400 | Missing Fields | `"Missing required fields: userId, productId, and projectName are required"` |
| 400 | Invalid Format | `"Invalid number format: {error details}"` |
| 400 | Duplicate Keys | `"Duplicate keys in form data"` |
| 500 | Processing Error | `"Error processing uploaded files: {error details}"` |
| 500 | Generation Failed | `"Failed to generate QR codes"` |

## Development Setup

### Prerequisites

- Java 17+
- Maven 3.8+

## Debug Levels
The system supports configurable debug output through the `DEBUG` environment variable to help with development and troubleshooting.

### Debug Level Configuration

| Debug Level | Environment Variable | Output Behavior |
|-------------|---------------------|-----------------|
| **Production** | `DEBUG=0` or unset | Silent operation - no debug output |
| **Basic Debug** | `DEBUG=1` | Essential debug information |
| **Verbose Debug** | `DEBUG=2` | Detailed processing information |

### Usage Examples

#### Production Environment (No Debug Output)
```bash
mvn function:run
```

## Generating Test QR Codes
You can generate test QR Codes by Setting the environment variable `TEST=1`.
This sets the cursor to 0 and adds a test flag to the generated codes
```bash
DEBUG=1 TEST=1 GOOGLE_APPLICATION_CREDENTIALS='/path/to/google-credentials.json' ./mvnw function:run
```

### Local Development

1. **Clone the repository**
```bash
git clone https://bitbucket.org/pesira-workspace/traceability.git
cd traceability
```

2. **Install dependencies**
```bash
./mvnw clean install
```

3. **Run locally**
```bash
DEBUG=1 \
GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account/credentials" \
./mvnw function:run
```

4. **Test the function**
```bash
curl --header "Content-Type: multipart/form-data" \
   --request POST \
   -F "user_id=user123" \
   -F "project_id=prod456" \
   -F "project_name=B4WE" \
   -F "location=SK" \
   -F "supplier_code=001" \
   -F "quantity=35" \
   -F "size=500" \
   -F "logo0=@test_logos/b4we.png" \
   -F "logo1=@test_logos/canada.png" \
   -F "logo2=@test_logos/ciat.png" \
   -F "logo3=@test_logos/image.png" \
   http://localhost:8080
```

### Testing

Run the test suite:

```bash
GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account/credentials" \
./mvnw test
```
