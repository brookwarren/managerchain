package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	groupID = "e49381ab-79d4-45e5-ab5a-7c68720f1e15" // Arctic Wolf MA group ID
)

func main() {
	// Retrieve command line arguments
	if len(os.Args) != 3 {
		log.Fatal("Usage: go run main.go <groupID> <managerEmail>")
	}
	managerEmail := os.Args[2]

	// Authentication setup
	client := getAuthenticatedClient()

	// Fetch all users from the specified group
	groupMembers := getGroupMembers(client, groupID)

	// Filter group members by the managerEmail recursively
	userHierarchy := map[string]string{} // map[email]managerEmail
	findReportsRecursively(client, managerEmail, groupMembers, userHierarchy)

	// Print the final report
	fmt.Println("EmailAddress, ManagerEmailAddress")
	for email, manager := range userHierarchy {
		fmt.Printf("%s, %s\n", email, manager)
	}
}

func getAuthenticatedClient() *msgraph.GraphServiceClient {
	tenantID := os.Getenv("TENANT_ID")
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	config := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID),
		Scopes:       []string{"https://graph.microsoft.com/.default"},
	}

	httpClient := config.Client(context.Background())
	client := msgraph.NewGraphServiceClient(httpClient)

	return client
}

func getGroupMembers(client *msgraph.GraphServiceClient, groupID string) []string {
	members, err := client.GroupsById(groupID).Members().Get(context.Background(), nil)
	if err != nil {
		log.Fatalf("Error fetching group members: %v", err)
	}

	var emails []string
	for _, member := range members.GetValue() {
		if user, ok := member.(*models.User); ok {
			emails = append(emails, *user.GetMail())
		}
	}
	return emails
}

func findReportsRecursively(client *msgraph.GraphServiceClient, managerEmail string, groupMembers []string, userHierarchy map[string]string) {
	// Query the manager's direct reports
	reports, err := client.UsersById(managerEmail).DirectReports().Get(context.Background(), nil)
	if err != nil {
		log.Printf("Error fetching reports for %s: %v", managerEmail, err)
		return
	}

	for _, report := range reports.GetValue() {
		if user, ok := report.(*models.User); ok {
			email := *user.GetMail()

			// Only process users who are members of the group
			if contains(groupMembers, email) && userHierarchy[email] == "" {
				// Add the user and their manager to the hierarchy
				userHierarchy[email] = managerEmail

				// Recursively find reports for the new user
				findReportsRecursively(client, email, groupMembers, userHierarchy)
			}
		}
	}
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if strings.EqualFold(v, item) {
			return true
		}
	}
	return false
}
