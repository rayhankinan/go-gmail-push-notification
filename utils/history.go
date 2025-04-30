package utils

import (
	"iter"

	"google.golang.org/api/gmail/v1"
)

func GetHistoryIter(srv *gmail.Service, user string, latestHistoryID uint64) iter.Seq2[*gmail.ListHistoryResponse, error] {
	return func(yield func(*gmail.ListHistoryResponse, error) bool) {
		nextPageToken := ""

		for {
			history, err := srv.Users.History.List(user).StartHistoryId(latestHistoryID).HistoryTypes("messageAdded", "labelAdded").PageToken(nextPageToken).Do()
			if !yield(history, err) {
				return
			}
			if history.NextPageToken == "" {
				return
			}

			nextPageToken = history.NextPageToken
		}
	}
}

func GetUniqueMessagesFromHistory(srv *gmail.Service, user string, labelIds []string, latestHistoryID uint64) ([]*gmail.Message, error) {
	messages := []*gmail.Message{}
	setID := make(map[string]struct{})

	for historyResponse, err := range GetHistoryIter(srv, user, latestHistoryID) {
		if err != nil {
			return nil, err
		}

		for _, history := range historyResponse.History {
			// Add messages that are added to the inbox
			for _, messageAdded := range history.MessagesAdded {
				if _, exists := setID[messageAdded.Message.Id]; !exists {
					messages = append(messages, messageAdded.Message)
					setID[messageAdded.Message.Id] = struct{}{}
				}
			}

			// Check if the label IDs we monitor is one of the labels being added
			for _, labelAdded := range history.LabelsAdded {
				if _, exists := setID[labelAdded.Message.Id]; Intersects(labelAdded.LabelIds, labelIds) && !exists {
					messages = append(messages, labelAdded.Message)
					setID[labelAdded.Message.Id] = struct{}{}
				}
			}
		}
	}

	return messages, nil
}
