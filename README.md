# SSM and S3 Server

This project provides an HTTP server using Go, which interacts with AWS Systems Manager (SSM) to fetch parameters and with S3 to fetch files. The server exposes two main endpoints: `/ssm` and `/s3`.

## Features

- Fetch decrypted parameters from AWS SSM.
- Fetch and serve files from AWS S3.
- Basic CI/CD pipeline using GitHub Actions for automatic builds and tests.

## Requirements

- Go 1.17 or later
- AWS CLI configured with necessary permissions
- Git installed

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

### Setting Up the GitHub Actions Pipeline

1. **Ensure the workflow file exists:**

    The workflow file `.github/workflows/ci.yml` should be present in your repository. If not, create it and paste the following content:

    ```yaml
    name: Go CI

    on:
      push:
        branches: [ main ]
      pull_request:
        branches: [ main ]

    jobs:
      build:
        runs-on: ubuntu-latest 

        steps:
          - name: Checkout code
            uses: actions/checkout@v2

          - name: Set up Go
            uses: actions/setup-go@v2
            with:
              go-version: 1.17

          - name: Install dependencies
            run: go mod tidy

          - name: Build
            run: go build -v ./...

          - name: Run tests
            run: go test -v ./...
    ```

2. **Push changes to the repository:**

    Commit and push your changes to trigger the pipeline:

    ```sh
    git add .
    git commit -m "Add CI pipeline for build and test"
    git push origin main
    ```

3. **Review pipeline runs:**

    Go to the `Actions` tab in your GitHub repository to view the status of the workflow runs.

## Contribution

Feel free to fork this repository and create pull requests. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the MIT License.

