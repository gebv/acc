package stripe

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/labstack/echo"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/webhook"
)

// If you are testing your webhook locally with the Stripe CLI you
// can find the endpoint's secret by running `stripe trigger`
// Otherwise, find your endpoint's secret in your webhook settings in the Developer Dashboard
var endpointSecret = "whsec_IDoODms1QL7LRHhEpmwRuwAmp3lejhKi"

// StripeWebhookHandler обработчик вебхука от stripe.
func (p *Provider) SberbankWebhookHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		const MaxBodyBytes = int64(65536)
		c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, MaxBodyBytes)
		payload, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
			c.Response().WriteHeader(http.StatusServiceUnavailable)
			return nil
		}

		// Pass the request body and Stripe-Signature header to ConstructEvent, along
		// with the webhook signing key.
		event, err := webhook.ConstructEvent(payload, c.Request().Header.Get("Stripe-Signature"),
			endpointSecret)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
			c.Response().WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
			return nil
		}

		// Unmarshal the event data into an appropriate struct depending on its Type
		switch event.Type {
		case "payment_intent.succeeded":
			var paymentIntent stripe.PaymentIntent
			err := json.Unmarshal(event.Data.Raw, &paymentIntent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
				c.Response().WriteHeader(http.StatusBadRequest)
				return nil
			}
			fmt.Println("PaymentIntent was successful!")
		case "payment_method.attached":
			var paymentMethod stripe.PaymentMethod
			err := json.Unmarshal(event.Data.Raw, &paymentMethod)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
				c.Response().WriteHeader(http.StatusBadRequest)
				return nil
			}
			fmt.Println("PaymentMethod was attached to a Customer!")
		// ... handle other event types
		default:
			fmt.Fprintf(os.Stderr, "Unexpected event type: %s\n", event.Type)
			fmt.Fprintf(os.Stderr, "Unexpected event type: %s\n", event.ID)
			fmt.Fprintf(os.Stderr, "Unexpected event type: %s\n", string(event.Data.Raw))
			c.Response().WriteHeader(http.StatusBadRequest)
			return nil
		}

		c.Response().WriteHeader(http.StatusOK)
		return nil
	}
}
