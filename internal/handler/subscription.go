package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/subscription/subscription.controller.ts

Routes + request/response shape:
- POST /api/v1/subscription/redeem-coupon (HandleRedeemCoupon)
  Request: no explicit request params
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/subscription/stripe/callback (HandleGetStripeCallback)
  Request: no explicit request params
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/subscription/stripe/checkout-session (HandleCreateStripeCheckoutSession)
  Request: no explicit request params
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleRedeemCoupon(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleGetStripeCallback(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, rootURL()+"/"+defaultLanguageCode+"/account/membership", http.StatusFound)
}

func HandleCreateStripeCheckoutSession(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusCreated, map[string]any{})
}
