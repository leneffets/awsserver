# AWS Server

This project provides an HTTP server using Go, which interacts with AWS Services. This project is useful for running a sidecar, to use original images without a hassle.

## Features

- Fetch decrypted parameters from AWS SSM.
- Fetch and serve files from AWS S3.
- Upload files to AWS S3.
- Fetch ECR authorization token.
- Fetch caller identity from AWS STS.
- Basic CI/CD pipeline using GitHub Actions for automatic builds and tests.

## Requirements

- Go 1.17 or later
- AWS CLI configured with necessary permissions
- Git installed

## Container Image

Get the Container Image

   ```sh
   docker pull ghcr.io/leneffets/awsserver:v1.0.0
   docker pull ghcr.io/leneffets/awsserver:latest
   ```

## Setup

1. **Clone the repository:**

    ```sh
    git clone git@github.com:USERNAME/REPO_NAME.git
    cd REPO_NAME
    ```

2. **Initialize the Go module:**

    ```sh
    go mod tidy
    ```

3. **Configure AWS credentials:**

    Ensure you have AWS credentials configured, typically in `~/.aws/credentials`.

## Running the Server

To start the HTTP server locally on port 3000, run the following command:
    
    # Port may be changed via Environment, default 3000
    export PORT=3000
    go run cmd/server/main.go


## Endpoints

### Fetch SSM Parameter

Fetch a decrypted parameter from AWS SSM.

- **URL:** `/ssm`
- **Method:** `GET`
- **Query Parameters:**
  - `name`: Name of the SSM parameter to fetch.
- **Example:**

    ```sh
    curl "http://localhost:3000/ssm?name=example_parameter"
    ```

### Put SSM Parameter

Put a parameter to AWS SSM.

- **URL:** `/ssm`
- **Method:** `POST`
- **Query Parameters:**
  - `name`: Name of the SSM parameter to put.
  - `value`: Value of the SSM parameter to put.
  - `type`: Type of the SSM parameter to put. (String, SecureString)
- **Example:**

    ```sh
    curl -X POST -d "name=/path/to/parameter&value=somevalue&type=String" http://localhost:3000/ssm
    ```

### Fetch S3 File

Fetch a file from an S3 bucket.

- **URL:** `/s3`
- **Method:** `GET`
- **Query Parameters:**
  - `bucket`: Name of the S3 bucket.
  - `key`: Key of the file in the S3 bucket.
- **Example:**

    ```sh
    curl "http://localhost:3000/s3?bucket=example-bucket&key=example-key"
    ```

### Upload S3 File

Upload a file to an S3 bucket.

- **URL:** `/s3`
- **Method:** `POST`
- **Query Parameters:**
  - `bucket`: Name of the S3 bucket.
  - `key`: Key of the file in the S3 bucket.
- **Example:**

    ```sh
    curl -X POST -F 'file=@/path/to/your/file' "http://localhost:3000/s3?bucket=example-bucket&key=example-key"
    ```

### Get ECR Login

Fetch an authorization token for ECR.

- **URL:** `/ecr/login`
- **Method:** `GET`
- **Example:**

    ```sh
    curl "http://localhost:3000/ecr/login" |  docker login --username AWS --password-stdin aws-account-id.dkr.ecr.eu-central-1.amazonaws.com
    ```

### Get Caller Identity

Fetch the caller identity from AWS STS.

- **URL:** `/sts`
- **Method:** `GET`
- **Example:**

    ```sh
    curl "http://localhost:3000/sts"
    ```

## Running Tests

To run the tests:

    go test -v ./...

## CI/CD Pipeline with GitHub Actions

This project uses GitHub Actions for continuous integration. The pipeline is defined in `.github/workflows/ci.yml` and performs the following actions on each push or pull request to the `main` branch:

- Checks out the code.
- Sets up the Go environment.
- Installs dependencies.
- Builds the project.
- Runs tests.

3. **Review pipeline runs:**

    Go to the `Actions` tab in your GitHub repository to view the status of the workflow runs.

## Contribution

Feel free to fork this repository and create pull requests. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the MIT License.

