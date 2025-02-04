package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	serviceusage "cloud.google.com/go/serviceusage/apiv1"
	serviceusagepb "cloud.google.com/go/serviceusage/apiv1/serviceusagepb"
)

// Service represents a simplified GCP service configuration.
type Service struct {
	Name          string `json:"name"`
	Title         string `json:"title"`
	Documentation string `json:"documentation,omitempty"`
	Domain        string `json:"domain,omitempty"`
	// FileName is not saved in JSON; it is computed for linking pages.
	FileName string `json:"-"`
}

func main() {
	// Command-line flags.
	crawlFlag := flag.Bool("crawl", false, "Crawl GCP service usage and save service details to services.json")
	generateFlag := flag.Bool("generate", false, "Generate HTML pages from saved services.json data")
	flag.Parse()

	if *crawlFlag && *generateFlag {
		log.Fatal("Please specify only one command: -crawl or -generate")
	}
	if !*crawlFlag && !*generateFlag {
		flag.Usage()
		os.Exit(1)
	}

	if *crawlFlag {
		if err := crawlServices(); err != nil {
			log.Fatalf("Crawl failed: %v", err)
		}
	} else if *generateFlag {
		if err := generateHTML(); err != nil {
			log.Fatalf("Generate failed: %v", err)
		}
	}
}

// crawlServices contacts the Service Usage API and writes a services.json file.
func crawlServices() error {
	ctx := context.Background()

	client, err := serviceusage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create service usage client: %v", err)
	}
	defer client.Close()

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		return fmt.Errorf("GCP_PROJECT_ID environment variable is required")
	}
	parent := fmt.Sprintf("projects/%s", projectID)

	// Map to hold unique services keyed by service name.
	servicesMap := make(map[string]map[string]interface{})

	// Function to call the API with the given filter.
	callAPI := func(filter string) error {
		req := &serviceusagepb.ListServicesRequest{
			Parent: parent,
			Filter: filter,
		}

		// Override the number of parts to use for the domain name.
		overrides := map[string]int{
			".cloud.goog": 3,
		}

		it := client.ListServices(ctx, req)
		for {
			resp, err := it.Next()
			if err != nil {
				// Break out if iteration is done.
				break
			}

			name := resp.Config.Name
			// If we've already seen this service, skip it.
			if _, exists := servicesMap[name]; exists {
				continue
			}

			svc := map[string]interface{}{
				"name":  name,
				"title": resp.Config.Title,
			}

			parts := strings.Split(name, ".")
			count := 2 // default to the last two parts
			for suffix, overrideCount := range overrides {
				if strings.HasSuffix(name, suffix) {
					count = overrideCount
					break
				}
			}
			if len(parts) >= count {
				svc["domain"] = strings.Join(parts[len(parts)-count:], ".")
			}

			if summary := resp.Config.Documentation.Summary; summary != "" {
				svc["documentation"] = summary
			}

			servicesMap[name] = svc
		}
		return nil
	}

	// First call: get enabled services.
	if err := callAPI("state:ENABLED"); err != nil {
		return fmt.Errorf("failed to get enabled services: %v", err)
	}

	// Second call: get disabled services.
	if err := callAPI("state:DISABLED"); err != nil {
		return fmt.Errorf("failed to get disabled services: %v", err)
	}

	// Create a slice from the map.
	var services []map[string]interface{}
	for _, svc := range servicesMap {
		services = append(services, svc)
	}

	// Sort the slice by the "name" field.
	sort.Slice(services, func(i, j int) bool {
		return services[i]["name"].(string) < services[j]["name"].(string)
	})

	jsonData, err := json.MarshalIndent(services, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile("services.json", jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write services.json: %v", err)
	}

	fmt.Println("Service catalog saved to services.json")
	return nil
}

// generateHTML reads services.json and produces HTML pages.
// Domain detail pages are written into the "domain" subfolder
// and service detail pages into the "service" subfolder.
func generateHTML() error {
	// Read and unmarshal services.json.
	data, err := os.ReadFile("services.json")
	if err != nil {
		return fmt.Errorf("failed to read services.json: %v", err)
	}

	var services []Service
	if err := json.Unmarshal(data, &services); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// For each service, compute Domain (if missing) and a sanitized FileName.
	for i, svc := range services {
		if svc.Domain == "" {
			parts := strings.Split(svc.Name, ".")
			if len(parts) > 1 {
				services[i].Domain = strings.Join(parts[len(parts)-2:], ".")
			} else {
				services[i].Domain = "misc"
			}
		}
		// Create a file-safe name (e.g., replace "/" with "-").
		services[i].FileName = strings.ReplaceAll(svc.Name, "/", "-")
	}

	// Group services by domain.
	domainMap := make(map[string][]Service)
	for _, svc := range services {
		domainMap[svc.Domain] = append(domainMap[svc.Domain], svc)
	}

	// Create a sorted list of domains.
	var domains []string
	for d := range domainMap {
		domains = append(domains, d)
	}
	sort.Strings(domains)

	// Ensure output directories exist.
	htmlDir := "html"
	domainDir := filepath.Join(htmlDir, "domain")
	serviceDir := filepath.Join(htmlDir, "service")
	if err := os.MkdirAll(htmlDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create html directory: %v", err)
	}
	if err := os.MkdirAll(domainDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create domain directory: %v", err)
	}
	if err := os.MkdirAll(serviceDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create service directory: %v", err)
	}

	// Copy style.css to the output directory.
	if err := copyFile("assets/style.css", filepath.Join(htmlDir, "style.css")); err != nil {
		log.Fatalf("Error copying style.css: %v", err)
	}

	// Parse all external templates.
	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse templates: %v", err)
	}

	// -----------------------------------
	// 1. Generate Home Page (index.html)
	// -----------------------------------
	homeData := struct {
		TotalServices int
	}{
		TotalServices: len(services),
	}
	homeFile := filepath.Join(htmlDir, "index.html")
	homeOut, err := os.Create(homeFile)
	if err != nil {
		return fmt.Errorf("failed to create home page: %v", err)
	}
	defer homeOut.Close()
	if err := tmpl.ExecuteTemplate(homeOut, "index.html", homeData); err != nil {
		return fmt.Errorf("failed to execute home template: %v", err)
	}
	log.Printf("Generated home page: %s", homeFile)

	// -----------------------------------
	// 2. Generate "Services" Page (services.html)
	// -----------------------------------
	servicesData := struct {
		Services []Service
	}{
		Services: services,
	}
	servicesFile := filepath.Join(htmlDir, "services.html")
	svcOut, err := os.Create(servicesFile)
	if err != nil {
		return fmt.Errorf("failed to create services page: %v", err)
	}
	defer svcOut.Close()
	if err := tmpl.ExecuteTemplate(svcOut, "services.html", servicesData); err != nil {
		return fmt.Errorf("failed to execute services template: %v", err)
	}
	log.Printf("Generated services page: %s", servicesFile)

	// -----------------------------------
	// 3. Generate "By Domain" Page (bydomain.html)
	// -----------------------------------
	// Create a slice of domain summary items.
	type DomainSummary struct {
		Domain string
		Count  int
		Link   string
	}
	var domainSummaries []DomainSummary
	for _, d := range domains {
		// Set the link to the file in the "domain" folder.
		domainSummaries = append(domainSummaries, DomainSummary{
			Domain: d,
			Count:  len(domainMap[d]),
			Link:   fmt.Sprintf("domain/domain-%s.html", urlSafe(d)),
		})
	}
	byDomainData := struct {
		Domains []DomainSummary
	}{
		Domains: domainSummaries,
	}
	byDomainFile := filepath.Join(htmlDir, "bydomain.html")
	byDomainOut, err := os.Create(byDomainFile)
	if err != nil {
		return fmt.Errorf("failed to create bydomain page: %v", err)
	}
	defer byDomainOut.Close()
	if err := tmpl.ExecuteTemplate(byDomainOut, "bydomain.html", byDomainData); err != nil {
		return fmt.Errorf("failed to execute bydomain template: %v", err)
	}
	log.Printf("Generated bydomain page: %s", byDomainFile)

	// -----------------------------------
	// 4. Generate Domain Detail Pages (in the domain folder)
	// -----------------------------------
	for _, domain := range domains {
		domainData := struct {
			Domain   string
			Services []Service
		}{
			Domain:   domain,
			Services: domainMap[domain],
		}
		domainFileName := fmt.Sprintf("domain-%s.html", urlSafe(domain))
		domainFilePath := filepath.Join(domainDir, domainFileName)
		f, err := os.Create(domainFilePath)
		if err != nil {
			log.Printf("Failed to create domain page for %s: %v", domain, err)
			continue
		}
		if err := tmpl.ExecuteTemplate(f, "domain.html", domainData); err != nil {
			log.Printf("Failed to execute domain template for %s: %v", domain, err)
			f.Close()
			continue
		}
		f.Close()
		log.Printf("Generated domain page for %s: %s", domain, domainFilePath)
	}

	// -----------------------------------
	// 5. Generate Service Detail Pages (in the service folder)
	// -----------------------------------
	for _, svc := range services {
		serviceFileName := fmt.Sprintf("%s.html", svc.FileName)
		serviceFilePath := filepath.Join(serviceDir, serviceFileName)
		f, err := os.Create(serviceFilePath)
		if err != nil {
			log.Printf("Failed to create service page for %s: %v", svc.Name, err)
			continue
		}
		if err := tmpl.ExecuteTemplate(f, "service.html", svc); err != nil {
			log.Printf("Failed to execute service template for %s: %v", svc.Name, err)
			f.Close()
			continue
		}
		f.Close()
		log.Printf("Generated service page for %s: %s", svc.Name, serviceFilePath)
	}

	fmt.Printf("HTML generation completed. Check the '%s' directory for output.\n", htmlDir)
	return nil
}

// urlSafe returns a version of the input string safe for use in URLs and file names.
func urlSafe(s string) string {
	s = strings.ReplaceAll(s, " ", "-")
	return strings.ToLower(s)
}

// copyFile copies a file from source to destination.
func copyFile(source, destination string) error {
	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}
