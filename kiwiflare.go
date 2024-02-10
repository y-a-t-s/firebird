package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"hash"
	"log"
	"math/rand"
	"runtime"
	"strconv"
	"time"

	"golang.org/x/net/html"
)

// KiwiFlare instance. A variant of Hashcash.
type kiwiFlare struct {
	kfParams
	workers []kfWorker
	hashes  chan kfResult
}

type kfParams struct {
	salt     string // Challenge salt from server.
	diff     uint32 // Difficulty level.
	patience uint32 // Time limit for answer in minutes.
}

type kfWorker struct {
	hasher hash.Hash // Sha256 hasher obj.
	nonce  uint32    // Brute-forced nonce appended to salt.
}

type kfResult struct {
	hash  []byte
	nonce uint32
}

func (kr kfResult) Result() ([]byte, uint32) {
	return kr.hash, kr.nonce
}

func getKFParams(root *html.Node) (kfParams, error) {
	if root == nil {
		panic("Challenge page reference is nil.")
	}

	parseAttr := func(av string) uint32 {
		tmp, err := strconv.Atoi(av)
		if err != nil {
			log.Fatal(err)
		}
		return uint32(tmp)
	}

	kp := kfParams{}

	attrs := root.Attr
	for _, a := range attrs {
		switch a.Key {
		case "data-sssg-challenge":
			kp.salt = a.Val
		case "data-sssg-difficulty":
			kp.diff = parseAttr(a.Val)
		case "data-sssg-patience":
			kp.patience = parseAttr(a.Val)
		}
	}

	return kp, nil
}

func (kf *kiwiFlare) generate() error {
	if len(kf.workers) == 0 {
		panic("No initialized thread workers found.")
	}

	// Brute force nonces until hashes channel receives valid answer.
	runWorker := func(w *kfWorker) {
		for {
			w.hasher.Write([]byte(fmt.Sprintf("%s%d", kf.salt, w.nonce)))
			kf.hashes <- kfResult{
				hash:  w.hasher.Sum(nil),
				nonce: w.nonce,
			}
			// Reset input for next iteration.
			w.hasher.Reset()
			w.nonce++
		}
	}

	for i := range kf.workers {
		go runWorker(&kf.workers[i])
	}

	return nil
}

func (kf *kiwiFlare) Solve() (kfResult, error) {
	duration, err := time.ParseDuration(fmt.Sprintf("%dm", kf.patience))
	if err != nil {
		// Fallback to common value.
		duration = time.Minute * 3
	}
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	kf.workers = make([]kfWorker, runtime.NumCPU())
	for i := range kf.workers {
		kf.workers[i] = kfWorker{sha256.New(), rand.Uint32()}
	}
	go kf.generate()

	// Initializes to all 0s. Used below for prefix checking.
	// Given difficulty is measured in number of leading 0 bits. Here, we want # of bytes.
	zeros := make([]byte, kf.diff/8)
	// TODO: Compare at bit level for technical correctness.

	// Loop until answer has been found.
	for {
		select {
		// Return nil result if timeout has expired.
		case <-ctx.Done():
			panic("KiwiFlare challenge timed out.")
		case h := <-kf.hashes:
			if bytes.HasPrefix(h.hash, zeros) {
				return h, nil
			}
		}
	}
}
