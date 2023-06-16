package appleTransaction

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
	"time"
)

// all failure apple statuses https://developer.apple.com/documentation/appstorereceipts/status

// Transaction - transaction data from apple.
type Transaction struct {
	ID        string
	InAppName string

	// SubscriptionExpireAt - Unix timestamp.
	// 0 if it's not subscribe inapp.
	SubscriptionExpireAt int64
}

// TransactionsByReceipt - retrieve all transactions by apple receipt.
//
// Apple status != 0 will return in error as string.
func TransactionsByReceipt(receipt, sharedPassword string) (res []Transaction, err error) {
	var req = appleQuery{
		ReceiptData: receipt,
		Password:    sharedPassword,
	}

	resp, err := req.query(false)
	if err != nil {
		return res, errors.Wrap(err, "apple query(sandbox:false)")
	}

	if resp.Status == 21007 {
		resp, err = req.query(true)
		if err != nil {
			return res, errors.Wrap(err, "apple query(sandbox:true)")
		}
	}

	if resp.Status != 0 {
		return res, errors.New(strconv.Itoa(resp.Status))
	}

	return resp.collectTransactions()
}

// appleQuery - json payload for apple
type appleQuery struct {
	// ReceiptData - base64 encoded receipt
	ReceiptData string `json:"receipt-data"`
	// Password - Shared secret
	Password string `json:"password,omitempty"`
}

func (q *appleQuery) query(sandbox bool) (res receiptData, err error) {
	var appStoreURL string

	if sandbox {
		appStoreURL = "https://sandbox.itunes.apple.com/verifyReceipt"
	} else {
		appStoreURL = "https://buy.itunes.apple.com/verifyReceipt"
	}

	// Encode json data for App Store
	buffer := new(bytes.Buffer)

	if err = json.NewEncoder(buffer).Encode(q); err != nil {
		return res, errors.Wrap(err, "failed Encode")
	}

	// Send receipt to App Store
	response, err := http.Post(appStoreURL, "application/json", buffer)
	if err != nil {
		return res, errors.Wrap(err, "failed http.Post")
	}

	defer func() { _ = response.Body.Close() }()

	if err = json.NewDecoder(response.Body).Decode(&res); err != nil {
		return res, errors.Wrap(err, "failed Decode response")
	}

	return
}

type latestReceipt struct {
	Quantity                    string `json:"quantity"`
	ProductID                   string `json:"product_id"`
	TransactionID               string `json:"transaction_id"`
	OriginalTransactionID       string `json:"original_transaction_id"`
	PurchaseDate                string `json:"purchase_date"`
	PurchaseDateMS              string `json:"purchase_date_ms"`
	PurchaseDatePST             string `json:"purchase_date_pst"`
	OriginalPurchaseDate        string `json:"original_purchase_date"`
	OriginalPurchaseDateMS      string `json:"original_purchase_date_ms"`
	OriginalPurchaseDatePST     string `json:"original_purchase_date_pst"`
	ExpiresDate                 string `json:"expires_date"`
	ExpiresDateMS               string `json:"expires_date_ms"`
	ExpiresDatePST              string `json:"expires_date_pst"`
	WebOrderLineItemID          string `json:"web_order_line_item_id"`
	IsTrialPeriod               string `json:"is_trial_period"`
	IsInIntroOfferPeriod        string `json:"is_in_intro_offer_period"`
	InAppOwnershipType          string `json:"in_app_ownership_type"`
	SubscriptionGroupIdentifier string `json:"subscription_group_identifier"`
}

type pendingRenewalInfo struct {
	ExpirationIntent       string `json:"expiration_intent"`
	AutoRenewProductID     string `json:"auto_renew_product_id"`
	IsInBillingRetryPeriod string `json:"is_in_billing_retry_period"`
	ProductID              string `json:"product_id"`
	OriginalTransactionID  string `json:"original_transaction_id"`
	AutoRenewStatus        string `json:"auto_renew_status"`
}

type receipt struct {
	ReceiptType                string  `json:"receipt_type"`
	AdamID                     int     `json:"adam_id"`
	AppItemID                  int     `json:"app_item_id"`
	BundleID                   string  `json:"bundle_id"`
	ApplicationVersion         string  `json:"application_version"`
	DownloadID                 int     `json:"download_id"`
	VersionExternalIdentifier  int     `json:"version_external_identifier"`
	ReceiptCreationDate        string  `json:"receipt_creation_date"`
	ReceiptCreationDateMS      string  `json:"receipt_creation_date_ms"`
	ReceiptCreationDatePST     string  `json:"receipt_creation_date_pst"`
	RequestDate                string  `json:"request_date"`
	RequestDateMS              string  `json:"request_date_ms"`
	RequestDatePST             string  `json:"request_date_pst"`
	OriginalPurchaseDate       string  `json:"original_purchase_date"`
	OriginalPurchaseDateMS     string  `json:"original_purchase_date_ms"`
	OriginalPurchaseDatePST    string  `json:"original_purchase_date_pst"`
	OriginalApplicationVersion string  `json:"original_application_version"`
	InApp                      []inApp `json:"in_app"`
}

type inApp struct {
	Quantity                string `json:"quantity"`
	ProductID               string `json:"product_id"`
	TransactionID           string `json:"transaction_id"`
	OriginalTransactionID   string `json:"original_transaction_id"`
	PurchaseDate            string `json:"purchase_date"`
	PurchaseDateMS          string `json:"purchase_date_ms"`
	PurchaseDatePST         string `json:"purchase_date_pst"`
	OriginalPurchaseDate    string `json:"original_purchase_date"`
	OriginalPurchaseDateMS  string `json:"original_purchase_date_ms"`
	OriginalPurchaseDatePST string `json:"original_purchase_date_pst"`
	ExpiresDate             string `json:"expires_date"`
	ExpiresDateMS           string `json:"expires_date_ms"`
	ExpiresDatePST          string `json:"expires_date_pst"`
	WebOrderLineItemID      string `json:"web_order_line_item_id"`
	IsTrialPeriod           string `json:"is_trial_period"`
	IsInIntroOfferPeriod    string `json:"is_in_intro_offer_period"`
	InAppOwnershipType      string `json:"in_app_ownership_type"`
}

// ReceiptData - ReceiptValidationResult contains validation result returned to client
type receiptData struct {
	Status             int                  `json:"status"`
	Environment        string               `json:"environment"`
	LatestReceiptInfo  []latestReceipt      `json:"latest_receipt_info"`
	PendingRenewalInfo []pendingRenewalInfo `json:"pending_renewal_info"`
	Receipt            receipt              `json:"receipt"`
	LatestReceipt      string               `json:"latest_receipt"`
}

// msToTime - string milliseconds to int unix time.
func msToTime(ms string) (int64, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return 0, err
	}

	return time.UnixMilli(msInt).Unix(), nil
}

// collectTransactions - will return transactions with unique transaction_id.
func (r *receiptData) collectTransactions() (res []Transaction, err error) {
	var unique = make(map[string]Transaction)

	for _, v := range r.LatestReceiptInfo {
		var expires int64

		if v.ExpiresDateMS != "" {
			expires, err = msToTime(v.ExpiresDateMS)
			if err != nil {
				return res, errors.Wrap(err, "msToTime1 fail")
			}
		}

		unique[v.TransactionID] = Transaction{
			ID:                   v.TransactionID,
			InAppName:            v.ProductID,
			SubscriptionExpireAt: expires,
		}
	}

	for _, v := range r.Receipt.InApp {
		var expires int64

		if v.ExpiresDateMS != "" {
			expires, err = msToTime(v.ExpiresDateMS)
			if err != nil {
				return res, errors.Wrap(err, "msToTime2 fail")
			}
		}

		unique[v.TransactionID] = Transaction{
			ID:                   v.TransactionID,
			InAppName:            v.ProductID,
			SubscriptionExpireAt: expires,
		}
	}

	for _, v := range unique {
		res = append(res, v)
	}

	return
}
