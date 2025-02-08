# gcp-service-catalog

A catalog of GCP services for easy browsing and filtering

**Website:** https://gcp-service-catalog.unitvectorylabs.com/

## Overview

**gcp-service-catalog** is a website that provides an organized and searchable catalog of the GCP service URLs. The content is automatically generated by crawling the GCP API, ensuring up-to-date and accurate information.

## How It Works

This application is written in Go for both the data collection and site generation processes. The workflow consists of the following steps:

1. **Data Collection:**
    - A GitHub Action [gcp-service-catalog-crawl.yml](https://github.com/UnitVectorY-Labs/gcp-service-catalog/blob/main/.github/workflows/gcp-service-catalog-crawl.yml) runs daily to crawl the GCP API.
    - It fetches all services, saving the data as JSON files in the repository in the [services.json](https://github.com/UnitVectorY-Labs/gcp-service-catalog/blob/main/services.json) file.
2. **Site Generation:**
    - Another GitHub Action [gcp-service-catalog-generate.yaml](https://github.com/UnitVectorY-Labs/gcp-service-catalog/blob/main/.github/workflows/gcp-service-catalog-generate.yaml) triggers upon updates to the `main` branch.
    - It generates static HTML pages from the JSON data using the Go application.
    - Search functionality is implemented using JavaScript client-side.
3. **Hosting:**
    - The website is hosted on GitHub Pages.
