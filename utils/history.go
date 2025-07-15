package utils

import (
	"iter"

	"google.golang.org/api/gmail/v1"
)

func GetHistoryIter(srv *gmail.Service, user string, latestHistoryID uint64) iter.Seq2[*gmail.ListHistoryResponse, error] {
	return func(yield func(*gmail.ListHistoryResponse, error) bool) {
		nextPageToken := ""

		for {
			history, err := srv.Users.History.List(user).StartHistoryId(latestHistoryID).LabelId("UNREAD").HistoryTypes("messageAdded").PageToken(nextPageToken).Do()
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

func GetMessagesFromHistory(srv *gmail.Service, user string, latestHistoryID uint64) ([]*gmail.Message, error) {
	messages := []*gmail.Message{}

	for historyResponse, err := range GetHistoryIter(srv, user, latestHistoryID) {
		if err != nil {
			return nil, err
		}

		for _, history := range historyResponse.History {
			// Add messages that are added to the inbox
			for _, messageAdded := range history.MessagesAdded {
				messages = append(messages, messageAdded.Message)
			}
		}
	}

	return messages, nil
}
