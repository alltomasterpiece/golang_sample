package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/logger"
	"github.com/kickback-app/api/pkg/models"
	"github.com/kickback-app/api/server/handlers"
	"github.com/kickback-app/api/utils"
)

func (s S) GetExpenses(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	res, err := s.ExpenseService.GetExpenses(c, kickbackID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	currUser := utils.CurrentUser(c).ID
	totalExpenses, expensesOwed, expensesOwes := []models.M{}, []models.M{}, []models.M{}
	totalOwed, totalCollected, totalOwes, totalPaid := 0.0, 0.0, 0.0, 0.0
	for _, exp := range res {
		associatedUsers := s.ExpenseService.AssociatedUserIDs(exp)
		userObjs := s.UserService.SummarizeUsers(c, associatedUsers)
		totalExpenses = append(totalExpenses, s.ExpenseService.ResolveLinks(c, exp))
		relation, assignees := s.ExpenseService.Contextualize(exp, currUser)
		for _, a := range assignees {
			if relation == models.ExpenseOwed {
				totalOwed += a.Amount
				if a.IsCompleted {
					totalCollected += a.Amount
				}
				expensesOwed = append(expensesOwed, a.ToMap(models.M{
					"_id":        exp.ID,
					"name":       exp.Name,
					"is_private": exp.IsPrivate,
					"assignee":   userObjs.Find(a.UserID),
				}))
			}
			if relation == models.ExpenseOwes {
				totalOwes += a.Amount
				if a.IsCompleted {
					totalPaid += a.Amount
				}
				expensesOwes = append(expensesOwes, a.ToMap(models.M{
					"_id":        exp.ID,
					"name":       exp.Name,
					"is_private": exp.IsPrivate,
					"created_by": userObjs.Find(exp.CreatedBy),
				}))
			}
		}
	}
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{
		"total_collected": totalCollected,
		"total_owed":      totalOwed,
		"total_paid":      totalPaid,
		"total_owes":      totalOwes,
		"expenses_owed":   expensesOwed,
		"expenses_owes":   expensesOwes,
		"total_expenses":  totalExpenses,
		"user":            s.UserService.SummarizeUsers(c, []string{currUser}).Find(currUser),
	})
}

func (s S) CreateExpense(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingBodyFieldError{Field: "parentId"})
		return
	}
	var expense models.Expense
	if err := json.NewDecoder(c.Request.Body).Decode(&expense); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	expense.ParentID = kickbackID
	expenseID, err := s.ExpenseService.CreateExpense(c, &expense)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	event, err := s.EventService.GetEvent(c, kickbackID)
	if err != nil {
		logger.Error(c, "unable to get event details: %v", err)
		handlers.EncodeError(c, err)
		return
	}
	s.doSendNotification(c, models.Notification{
		Type:     models.ExpenseCreated,
		Channels: []string{"push"},
		To:       assigneesToUserIDs(expense.Assignees, false),
		Title:    "You've been assigned a new expense",
		Body:     fmt.Sprintf("%v has been assigned to you for Kickback %v", expense.Name, event.Name),
		Data: map[string]string{
			"eventId": kickbackID,
		},
	})
	logger.Info(c, "created expense %s within kickback %s", expenseID, expense.ParentID)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"expenseId": expenseID})
}

func (s S) GetExpense(c *gin.Context) {
	expenseID := c.Param("expenseId")
	if expenseID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: "expenseId"})
		return
	}
	expense, err := s.ExpenseService.GetExpense(c, expenseID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "retrieved expense %s", expenseID)
	handlers.EncodeSuccess(c, http.StatusOK, expense)
}

func (s S) UpdateExpense(c *gin.Context) {
	expenseID := c.Param("expenseId")
	if expenseID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: "expenseId"})
		return
	}
	var input models.ExpenseUpdates
	if err := json.NewDecoder(c.Request.Body).Decode(&input); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.ExpenseService.UpdateExpense(c, expenseID, &input)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	/*
	 * @todo can we isolate this to "UpdateExpenseAssignee" to avoid duplicate code?
	 *
	 */
	// get updated expense for notification data
	expense, err := s.ExpenseService.GetExpense(c, expenseID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	// get event information for notification data
	event, err := s.EventService.GetEvent(c, expense.ParentID)
	if err != nil {
		logger.Error(c, "unable to get event details: %v", err)
		handlers.EncodeError(c, err)
		return
	}
	if expense.CreatedBy == utils.CurrentUser(c).ID && input.Assignees != nil {
		// the charger is marking as paid so notify chargee
		s.doSendNotification(c, models.Notification{
			Type:     models.ExpenseUpdated,
			Channels: []string{"push"},
			To:       assigneesToUserIDs(*input.Assignees, true),
			Title:    "Your assigned expense has been updated",
			Body:     fmt.Sprintf("%v has been marked as completed in your Kickback %v", expense.Name, event.Name),
			Data: map[string]string{
				"eventId": expense.ParentID,
			},
		})
	}

	if expense.CreatedBy != utils.CurrentUser(c).ID && input.Assignees != nil {
		// the chargee is marking as paid so notify charger
		user, err := s.UserService.GetUserByID(c, utils.CurrentUser(c).ID)
		if err != nil {
			logger.Error(c, "unable to get user details: %v", err)
			handlers.EncodeError(c, err)
			return
		}
		s.doSendNotification(c, models.Notification{
			Type:     models.ExpenseUpdated,
			Channels: []string{"push"},
			To:       []string{expense.CreatedBy},
			Title:    "Your expense has been updated",
			Body:     fmt.Sprintf("%v marked your expense %v as completed in your Kickback %v", user.Name(), expense.Name, event.Name),
			Data: map[string]string{
				"eventId": expense.ParentID,
			},
		})
	}
	logger.Info(c, "successfully updated expense %s", expenseID)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"expense": expense})
}

func (s S) DeleteExpense(c *gin.Context) {
	expenseID := c.Param("expenseId")
	if expenseID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: "expenseId"})
		return
	}
	// get updated expense for notification data
	expense, err := s.ExpenseService.GetExpense(c, expenseID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	// get event information for notification data
	event, err := s.EventService.GetEvent(c, expense.ParentID)
	if err != nil {
		logger.Error(c, "unable to get event details: %v", err)
		handlers.EncodeError(c, err)
		return
	}
	// do the deletion
	err = s.ExpenseService.DeleteExpense(c, expenseID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	if expense.CreatedBy == utils.CurrentUser(c).ID {
		// the charger is marking as paid so notify chargee
		s.doSendNotification(c, models.Notification{
			Type:     models.ExpenseDeleted,
			Channels: []string{"push"},
			To:       assigneesToUserIDs(expense.Assignees, false),
			Title:    "Your assigned expense has been deleted",
			Body:     fmt.Sprintf("%v has been deleted in your Kickback %v", expense.Name, event.Name),
			Data: map[string]string{
				"eventId": expense.ParentID,
			},
		})
	}

	if expense.CreatedBy != utils.CurrentUser(c).ID {
		// the chargee is marking as paid so notify charger
		user, err := s.UserService.GetUserByID(c, utils.CurrentUser(c).ID)
		if err != nil {
			logger.Error(c, "unable to get user details: %v", err)
			handlers.EncodeError(c, err)
			return
		}
		s.doSendNotification(c, models.Notification{
			Type:     models.ExpenseDeleted,
			Channels: []string{"push"},
			To:       []string{expense.CreatedBy},
			Title:    "Your expense has been deleted",
			Body:     fmt.Sprintf("%v marked your expense %v as completed in your Kickback %v", user.Name(), expense.Name, event.Name),
			Data: map[string]string{
				"eventId": expense.ParentID,
			},
		})
	}
	logger.Info(c, "successfully deleted expense %s", expenseID)
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

/*
 * this same functionality can be achieved from using UpdateExpense directly, but this method
 * offers a more tailored route
 *
 */
func (s S) UpdateExpenseAssignee(c *gin.Context) {
	expenseID := c.Param("expenseId")
	if expenseID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: "expenseId"})
		return
	}
	var input struct {
		IsComepleted bool   `json:"is_completed"`
		Assignee     string `json:"assignee"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&input); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.ExpenseService.UpdateAssignee(c, expenseID, input.Assignee, input.IsComepleted)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	expense, err := s.ExpenseService.GetExpense(c, expenseID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	// get event information for notification data
	event, err := s.EventService.GetEvent(c, expense.ParentID)
	if err != nil {
		logger.Error(c, "unable to get event details: %v", err)
		handlers.EncodeError(c, err)
		return
	}
	if expense.CreatedBy == utils.CurrentUser(c).ID && input.IsComepleted {
		// the charger is marking as paid so notify chargee
		s.doSendNotification(c, models.Notification{
			Type:     models.ExpenseUpdated,
			Channels: []string{"push"},
			To:       []string{input.Assignee},
			Title:    "Your assigned expense has been updated",
			Body:     fmt.Sprintf("%v has been marked as completed in your Kickback %v", expense.Name, event.Name),
			Data: map[string]string{
				"eventId": expense.ParentID,
			},
		})
	}

	if expense.CreatedBy != utils.CurrentUser(c).ID && input.IsComepleted {
		// the chargee is marking as paid so notify charger
		user, err := s.UserService.GetUserByID(c, utils.CurrentUser(c).ID)
		if err != nil {
			logger.Error(c, "unable to get user details: %v", err)
			handlers.EncodeError(c, err)
			return
		}
		s.doSendNotification(c, models.Notification{
			Type:     models.ExpenseUpdated,
			Channels: []string{"push"},
			To:       []string{expense.CreatedBy},
			Title:    "Your expense has been updated",
			Body:     fmt.Sprintf("%v marked your expense %v as completed in your Kickback %v", user.Name(), expense.Name, event.Name),
			Data: map[string]string{
				"eventId": expense.ParentID,
			},
		})
	}
	logger.Info(c, "successfully updated users expenses status to %v", input.IsComepleted)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"expense": expense})
}

func assigneesToUserIDs(assignees []models.ExpenseAssignee, onlyCompleted bool) []string {
	userIDs := []string{}
	for _, assignee := range assignees {
		if onlyCompleted && !assignee.IsCompleted {
			continue
		}
		userIDs = append(userIDs, assignee.UserID)
	}
	return userIDs
}
