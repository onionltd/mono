package main

import "net/http"

type oopsSet map[string]oopsMessages

type oopsMessages map[int]string

func (m oopsMessages) Get(id int) string {
	val, ok := m[id]
	if !ok {
		return m[0]
	}
	return val
}

var oopsies = oopsSet{
	"/links/oops/:id": {
		0:                              "Hey! You made that up!",
		69:                             "Haha, funny.",
		1337:                           "Look at you, hacker. A pathetic creature of meat and bone. Panting and sweating as you run through my corridors. How can you challenge a perfect immortal machine?",
		http.StatusBadRequest:          "This doesn't look like a valid link.",
		http.StatusNotFound:            "Your link does not belong to any service vworp! can recognize.",
		http.StatusNotAcceptable:       "Haha, so meta.",
		http.StatusInternalServerError: "Hmm... Something has broken down but don't worry it's not your fault.",
	},
	"/links/:fp/oops/:id": {
		0:                              "Hey! You made that up!",
		69:                             "Haha, funny.",
		1337:                           "Look at you, hacker. A pathetic creature of meat and bone. Panting and sweating as you run through my corridors. How can you challenge a perfect immortal machine?",
		http.StatusNotFound:            "I can't find that link",
		http.StatusInternalServerError: "Hmm... Something has broken down but don't worry it's not your fault.",
	},
	"/t/oops/:id": {
		0:                              "Hey! You made that up!",
		69:                             "Haha, funny.",
		1337:                           "Look at you, hacker. A pathetic creature of meat and bone. Panting and sweating as you run through my corridors. How can you challenge a perfect immortal machine?",
		http.StatusNotFound:            "I can't find that link",
		http.StatusInternalServerError: "Hmm... Something has broken down but don't worry it's not your fault.",
	},
	"/to/oops/:id": {
		0:                              "Hey! You made that up!",
		69:                             "Haha, funny.",
		1337:                           "Look at you, hacker. A pathetic creature of meat and bone. Panting and sweating as you run through my corridors. How can you challenge a perfect immortal machine?",
		http.StatusNotFound:            "I can't find that link",
		http.StatusInternalServerError: "Hmm... Something has broken down but don't worry it's not your fault.",
	},
}
