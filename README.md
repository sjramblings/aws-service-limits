# AWS Service Limits

This tool fetches and displays service quotas for AWS services.

## Features

- Query AWS service quotas and their current usage.
- Supports multiple output formats including tables, CSV, Markdown, and JSON.
- Can exclude items with a usage value of "Not Available".
- Lists all AWS services supported by the AWS Service Quota API.
- Color-coded display to easily spot "Not Available" usages.

## Installation

```bash
curl -L -o aws-service-limits https://github.com/sjramblings/aws-service-limits/releases/download/v0.1.0/aws-service-limits.linux-amd64

```

## Basic Usage

```bash
aws-service-limits
```

This will fetch and display AWS service quotas for EC2.

```bash
aws-service-limits --servicecode s3
```

This will fetch and display AWS service quotas for a specific AWS service (e.g., S3):

```bash
aws-service-limits --list-services
```

Lists all the services supported by the AWS Service Quota API:

## Commandline Flags

```bash
--timeframe: Timeframe for the CloudWatch query in hours. Options: 1, 24, 48, 72, etc. Default is 1 hour.
--servicecode: The AWS Service Code to query. Default is 'ec2'.
--format: Output format. Options: table (default), csv, markdown, json.
--exclude-na: Exclude items with a usage value of 'Not Available'.
--list-services: List all the services supported by the AWS Service Quota API and exit.
```

## Contributing

Feel free to open issues or PRs if you find any issues or have feature requests.
