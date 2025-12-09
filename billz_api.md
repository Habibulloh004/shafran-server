# Billz API (BILLZ 1) testing guide

Use this to exercise the BILLZ JSON-RPC API from https://api.billz.uz/docs/#/methods. All calls are `POST` with a JSON-RPC 2.0 body.

- Base URL: `https://api.billz.uz`
- Endpoints: `/v1/`, `/v2/`, `/v3/` (version depends on method)
- Headers: `Content-Type: application/json`, `Authorization: Bearer <jwt>`
- JWT: use the token you shared `c30fd0b08c69a72c3c20799e6dd7c30ad7e0026c0d7541f3419929d781795ed2d90e8e9dc2fc5314bb7adb0bd1b2d2511b650a1bbacbc86bfc49416f5e4e22bfc1c9465f66478d63878e7991453ff3a75ce45e376ebb828c402a10ff8ccaea73ed74d11ef92490dc99fd30544cc7820cf05697534e2136dc`
- Time values are ISO8601 strings (e.g., `2023-01-01T00:00:00Z`)

## Postman collection
1) Import `billz_api.postman_collection.json` in Postman.  
2) The collection sets variables for `billz_base_url` (`https://api.billz.uz`) and `billz_jwt` (the token above); update the token there when it rotates.  
3) Every request body already follows the JSON-RPC shape; hit Send to test.  
4) Responses return either a `result` object or an `error` object per JSON-RPC 2.0.

## curl template
```bash
export BILLZ_JWT='c30fd0b08c69a72c3c20799e6dd7c30ad7e0026c0d7541f3419929d781795ed2d90e8e9dc2fc5314bb7adb0bd1b2d2511b650a1bbacbc86bfc49416f5e4e22bfc1c9465f66478d63878e7991453ff3a75ce45e376ebb828c402a10ff8ccaea73ed74d11ef92490dc99fd30544cc7820cf05697534e2136dc'
curl -X POST "https://api.billz.uz/v1/" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${BILLZ_JWT}" \
  -d '{
    "jsonrpc": "2.0",
    "method": "products.get",
    "params": {
      "LastUpdatedDate": "2018-03-21T18:19:25Z",
      "WithProductPhotoOnly": 0,
      "IncludeEmptyStocks": 0
    },
    "id": "products-get"
  }'
```

## Endpoint notes (see collection for full sample bodies)
- `products.get` (v1): filters by `LastUpdatedDate`, `WithProductPhotoOnly`, `FullSizePhoto`, `IncludeEmptyStocks`, `ProductIds`, `offices`; returns products with prices, qty, office breakdown, image URLs.
- `catalog.get` (v2): paginated catalog with `PerPage`, `Page`, optional `Sort`, `Filter` (statuses, offices, names, vendor codes, barcodes) and `Price` range; returns `total`, `page`, `results`, aggregations.

- `orders.create` (v1): create sale; fields `orderID`, `dateCreated`, `datePaid`, `paymentMethod`, `subTotalPrice`, `discountAmount`, `totalPrice`, `parked`, `products` (per-item office/product IDs, qty, prices).  
- `orders.create` (v3): same payload plus `clientID` (required when `paymentMethod` is `cashbackBalance`); endpoint `/v3/`.

- `client.get` (v1): fetch a client by `clientId` with optional `beginDate`, `endDate`, `lastUpdateDate`; returns client info, cards, balances, transactions, transaction details, payments.  
- `client.search` (v1): search by `phoneNumber` or `email` (phone takes priority); same response shape as `client.get`.

- Reports (v1 unless noted):  
  - `reports.sales`: sales lines for a date range with quantity, prices, margins, and product attributes.  
  - `reports.transfers`: transfer lines by `dateBegin`, `dateEnd`, `currency`; v2 (endpoint `/v2/`) also returns product IDs and timestamps.  
  - `reports.imports`: imports by date range; v2 (endpoint `/v2/`) adds product IDs and import type.  
  - `reports.consolidated`: store-level KPIs (revenue, discounts, returns, profit, inventory).  
  - `reports.cheques`: receipt-level summary grouped by payment types; optional `Office` array.  
  - `reports.writeoffs`: write-off lines filtered by `dateBegin`, `dateEnd`, `officeIDs`.  
  - `reports.inventory`: inventory results for a specific `stockList` ID.  
  - `reports.fin.transactions`: financial transactions by date range with account and currency info.  
  - `reports.clients.stats`: counts of total/new/returning clients per store for the period.

- Imports / stock intake:  
  - `import.create` (v1): create an import for a single office with `officeID` and `items` (id/barcode, sku, category, name, price, quantity); returns `ImportId`.  
  - `import.createWithOffice` (v1): create import where each item carries `details` listing `officeID` and `quantity`.

## Tips for sending and reading data
- Always include `jsonrpc`, `method`, `params`, and `id` in the request body.
- Monetary fields are numeric; set `currency` as `"UZS"` or `"USD"` where applicable.
- Large filters (`IncludeEmptyStocks` on `products.get`) can return heavy payloads—use pagination/filters when possible.
- Errors come back under the `error` key with code/message; successful calls use `result`.
