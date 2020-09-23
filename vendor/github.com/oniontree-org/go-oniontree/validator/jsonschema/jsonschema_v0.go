package jsonschema

const V0 = `
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "http://json-schema.org/draft-07/schema#",
  "title": "OnionTree service file schema",
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "minLength": 1
    },
    "description": {
      "type": "string"
    },
    "urls": {
      "type": "array",
      "minItems": 1,
      "uniqueItems": true,
      "items": {
        "type": "string",
        "pattern": "^http[s]?://.*\\.onion$"
      }
    },
    "public_keys": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": {
            "type": "string",
            "minLength": 16
          },
          "user_id": {
            "type": "string",
            "minLength": 1
          },
          "fingerprint": {
            "type": "string",
            "minLength": 40
          },
          "value": {
            "type": "string"
          }
        },
        "required": [
          "id",
          "user_id",
          "fingerprint",
          "value"
        ]
      }
    }
  },
  "required": [
    "name",
    "urls"
  ]
}`
