// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package registry_domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/registry/registry_schema"
)

const (
	// ReleaseStatePublishing marks a release lease whose owner is still uploading records
	// and bytes. Another node that finds this state waits, or takes it over once the owner's
	// heartbeat is older than publishingTakeoverTTL.
	ReleaseStatePublishing = "publishing"

	// ReleaseStatePublished marks a release lease whose records are fully written and are
	// safe for a foreign-release node to serve by storage key.
	ReleaseStatePublished = "published"

	// publishingTakeoverTTL is how stale a publishing lease's heartbeat must be before
	// another node deletes it and re-claims the publish.
	publishingTakeoverTTL = 5 * time.Minute

	// publishHeartbeatInterval is how often a long publish advances its own lease heartbeat
	// between artefact layers, which bounds takeover false positives regardless of seed
	// size.
	publishHeartbeatInterval = 30 * time.Second

	// logFieldRelease is the structured-log attribute key for a release identifier.
	logFieldRelease = "release"
)

const (
	// PublishOutcomeUnsupported means the overlay does not provide ReleasePublisher (a
	// single binary with no shared database), so nothing was published and the base serves
	// everything.
	PublishOutcomeUnsupported PublishOutcome = iota

	// PublishOutcomePublished means this caller claimed the release and wrote its layers.
	PublishOutcomePublished

	// PublishOutcomeAlreadyPublished means another node of the same release already
	// published an identical payload, so this call did a single lease read and no writes.
	PublishOutcomeAlreadyPublished

	// PublishOutcomeInProgress means another node holds the publishing lease; this caller
	// left it to finish and will serve from its own base until then.
	PublishOutcomeInProgress
)

var (
	// ErrReleaseDigestConflict marks a publish that found the same release id already
	// published with a different payload digest.
	ErrReleaseDigestConflict = errors.New("release digest conflict")
)

// ReleaseLease is a release's publish-lease row: the coordination record that lets two
// releases coexist on one shared overlay during a canary or rolling deploy, and that a
// reaper consults to retire a release whose nodes have all gone away.
type ReleaseLease struct {
	// ReleaseID identifies the release.
	ReleaseID string

	// PublishDigest is the digest of the payload the release published; a mismatch under one
	// release id means two different binaries claimed the same release, which is a hard
	// error.
	PublishDigest string

	// State is ReleaseStatePublishing or ReleaseStatePublished.
	State string

	// FirstSeenAt is when the release was first claimed, in Unix seconds.
	FirstSeenAt int64

	// PublishedAt is when the release finished publishing, in Unix seconds.
	PublishedAt int64

	// HeartbeatAt is the release's most recent heartbeat, in Unix seconds.
	HeartbeatAt int64

	// RetiredAt is when the release was retired, in Unix seconds, or zero.
	RetiredAt int64
}

// ReleasePublisher is the capability a shared, layer-aware overlay backend provides so a
// release can publish its artefact layers and coordinate a publish lease.
//
// Only the SQL DAL implements it; the single-node otter overlay and the immutable
// snapshot base do not, so a deployment without a shared database publishes nothing and
// serves entirely from its own base. The methods are the lock-free publish primitives: an
// artefact layer is written whole with InsertArtefactLayerIfAbsent (idempotent across
// racing nodes), and the lease is a single coordination row claimed with ClaimRelease.
type ReleasePublisher interface {
	// InsertArtefactLayerIfAbsent writes one immutable release layer of an artefact when its
	// (id, release) is not already present.
	//
	// A row is written only when it is absent, so the caller increments blob reference counts
	// exactly once per published layer.
	//
	// Takes artefact (*registry_dto.ArtefactMeta) which is the artefact layer to write.
	//
	// Returns bool which reports whether a row was newly written.
	// Returns error when the write fails.
	InsertArtefactLayerIfAbsent(ctx context.Context, artefact *registry_dto.ArtefactMeta) (bool, error)

	// DeleteArtefactLayersForRelease removes every artefact layer of a release, retiring it.
	//
	// Takes releaseID (string) which identifies the release to retire.
	//
	// Returns error when the delete fails.
	DeleteArtefactLayersForRelease(ctx context.Context, releaseID string) error

	// ReclaimArtefactLayersForRelease removes every artefact layer of a release and returns
	// the deleted layers, so the caller can decrement the blob references those layers held.
	//
	// The delete-and-return is atomic per statement: exactly one of two racing reapers
	// observes the rows and decrements.
	//
	// Takes releaseID (string) which identifies the release to reclaim.
	//
	// Returns []*registry_dto.ArtefactMeta which are the artefact layers that were deleted.
	// Returns error when the reclaim fails.
	ReclaimArtefactLayersForRelease(ctx context.Context, releaseID string) ([]*registry_dto.ArtefactMeta, error)

	// ClaimRelease claims publishing rights for a release.
	//
	// This caller wins the claim when it inserts the lease row, and loses when another caller
	// already holds it.
	//
	// Takes releaseID (string) which identifies the release to claim.
	// Takes publishDigest (string) which fingerprints the content being published.
	// Takes firstSeenAt (int64) which is when the release was first claimed, in Unix seconds.
	// Takes heartbeatAt (int64) which is the first heartbeat to stamp, in Unix seconds.
	//
	// Returns bool which reports whether this caller won the claim.
	// Returns error when the claim fails.
	ClaimRelease(ctx context.Context, releaseID, publishDigest string, firstSeenAt, heartbeatAt int64) (bool, error)

	// GetRelease returns a release lease and whether it exists.
	//
	// Takes releaseID (string) which identifies the release.
	//
	// Returns ReleaseLease which is the stored lease, or the zero value when it is absent.
	// Returns bool which reports whether the lease exists.
	// Returns error when the lookup fails.
	GetRelease(ctx context.Context, releaseID string) (ReleaseLease, bool, error)

	// MarkReleasePublished flips a release lease to published and stamps its timestamps.
	//
	// Takes releaseID (string) which identifies the release.
	// Takes publishedAt (int64) which is when the release finished publishing, in Unix seconds.
	// Takes heartbeatAt (int64) which is the heartbeat to stamp, in Unix seconds.
	//
	// Returns error when the update fails.
	MarkReleasePublished(ctx context.Context, releaseID string, publishedAt, heartbeatAt int64) error

	// HeartbeatRelease advances a release's heartbeat when the new value is more recent.
	//
	// The update is monotonic, so an out-of-order heartbeat cannot rewind a fresher one.
	//
	// Takes releaseID (string) which identifies the release.
	// Takes heartbeatAt (int64) which is the new heartbeat, in Unix seconds.
	//
	// Returns error when the update fails.
	HeartbeatRelease(ctx context.Context, releaseID string, heartbeatAt int64) error

	// ListExpiredReleases returns published releases whose heartbeat predates the cutoff,
	// excluding the caller's own release so a node never reaps the release it is serving.
	//
	// Takes cutoff (int64) which is the heartbeat cutoff, in Unix seconds.
	// Takes ownRelease (string) which is the caller's own release, which is never listed.
	//
	// Returns []string which are the identifiers of the expired releases.
	// Returns error when the listing fails.
	ListExpiredReleases(ctx context.Context, cutoff int64, ownRelease string) ([]string, error)

	// DeleteReleaseLease removes a release lease row, so a retired release can be re-claimed
	// by a later deploy and the reaper's expiry listing converges.
	//
	// Takes releaseID (string) which identifies the release.
	//
	// Returns error when the delete fails.
	DeleteReleaseLease(ctx context.Context, releaseID string) error

	// DeleteStalePublishingLease removes a publishing lease whose heartbeat predates
	// staleBefore, so a publish that died mid-flight can be re-claimed by another node.
	//
	// Takes releaseID (string) which identifies the release.
	// Takes staleBefore (int64) which is the heartbeat cutoff, in Unix seconds.
	//
	// Returns error when the delete fails.
	DeleteStalePublishingLease(ctx context.Context, releaseID string, staleBefore int64) error
}

// BlobReplicator copies one variant's bytes (and its chunks' bytes) into the shared blob
// store when they are not already there, so a foreign-release node that resolves the
// variant's record can also serve its bytes. Implementations must be idempotent: publish
// may invoke it for content another node already replicated.
type BlobReplicator func(ctx context.Context, variant *registry_dto.Variant) error

// publishConfig carries the optional behaviours of one PublishRelease call.
type publishConfig struct {
	// replicateBlob copies a variant's bytes into the shared store before its record is
	// written, or nil when the deployment's bytes are already shared.
	replicateBlob BlobReplicator
}

// PublishOption configures optional PublishRelease behaviour.
type PublishOption func(*publishConfig)

// WithBlobReplicator makes the publish copy each variant's bytes into the shared blob
// store before writing its record.
//
// Takes replicate (BlobReplicator) which copies one variant's bytes when absent.
//
// Returns PublishOption which enables byte replication.
func WithBlobReplicator(replicate BlobReplicator) PublishOption {
	return func(config *publishConfig) {
		config.replicateBlob = replicate
	}
}

// PublishOutcome reports what a PublishRelease call did, so the boot path can log it and
// gate a readiness probe without inspecting the lease itself. Its values are declared
// with the package constants above so that all const declarations precede the package
// variables.
type PublishOutcome int

// String renders a PublishOutcome for logging.
//
// Returns string which names the outcome.
func (o PublishOutcome) String() string {
	switch o {
	case PublishOutcomeUnsupported:
		return "unsupported"
	case PublishOutcomePublished:
		return "published"
	case PublishOutcomeAlreadyPublished:
		return "already-published"
	case PublishOutcomeInProgress:
		return "in-progress"
	default:
		return "unknown"
	}
}

// SeedDigest computes a stable content digest over a set of artefact layers, independent
// of their order, so two nodes of one release compute the same digest and a different
// build under the same release id computes a different one.
//
// Takes artefacts ([]*registry_dto.ArtefactMeta) which are the release's artefact layers.
//
// Returns string which is the hex-encoded digest.
func SeedDigest(artefacts []*registry_dto.ArtefactMeta) string {
	payloads := make([]string, 0, len(artefacts))
	for _, artefact := range artefacts {
		if artefact == nil {
			continue
		}
		payloads = append(payloads, string(registry_schema.BuildArtefactMeta(artefact)))
	}
	slices.Sort(payloads)

	hasher := sha256.New()
	for _, payload := range payloads {
		_, _ = hasher.Write([]byte(payload))
		_, _ = hasher.Write([]byte{0})
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// PublishRelease publishes a release's artefact layers into a shared overlay using the
// lock-free claim-then-insert protocol, so a canary or rolling deploy can put two
// releases on one shared database and each node can serve the other release's assets by
// storage key.
//
// The protocol: claim the release lease; if this caller wins, clone and stamp every
// artefact with the release id and write it as an immutable (id, release) layer with
// InsertArtefactLayerIfAbsent, incrementing blob reference counts only for layers
// actually inserted, so two nodes racing to publish the same release each write
// byte-identical rows and reference counts never double. A caller that does not win the
// claim inspects the existing lease: a published lease with a matching digest is a
// completed publish (zero work), a published lease with a different digest is a hard
// error wrapped with ErrReleaseDigestConflict (two binaries under one release id), and a
// still-publishing lease is left for its owner unless the owner's heartbeat is older than
// publishingTakeoverTTL, in which case the stale lease is deleted and the publish
// re-claimed (the owner died mid-flight).
//
// Publishing is an optimisation for multi-node serving, never a correctness requirement:
// a node always serves its own release from its own immutable base, so a failed or
// skipped publish only degrades cross-release serving. Callers therefore treat a returned
// error as non-fatal, except a digest conflict, which is a deploy misconfiguration to
// surface loudly.
//
// Takes overlay (MetadataStore) which is the shared writable overlay; publishing is a
// no-op unless it also implements ReleasePublisher.
// Takes releaseID (string) which identifies the release being published; it must be
// non-empty.
// Takes digest (string) which is the release payload digest (see SeedDigest).
// Takes artefacts ([]*registry_dto.ArtefactMeta) which are the release's artefact layers.
// Takes now (time.Time) which is the claim and heartbeat time.
//
// Returns PublishOutcome which describes what happened.
// Returns error when the release id is empty, a digest conflict is detected, or a store
// call fails.
func PublishRelease(
	ctx context.Context,
	overlay MetadataStore,
	releaseID, digest string,
	artefacts []*registry_dto.ArtefactMeta,
	now time.Time,
	options ...PublishOption,
) (PublishOutcome, error) {
	config := publishConfig{}
	for _, option := range options {
		option(&config)
	}
	ctx, l := logger_domain.From(ctx, log)

	publisher, ok := overlay.(ReleasePublisher)
	if !ok {
		return PublishOutcomeUnsupported, nil
	}
	if releaseID == "" {
		return PublishOutcomeUnsupported, errors.New("cannot publish a release with an empty release id")
	}

	nowSeconds := now.Unix()
	won, err := publisher.ClaimRelease(ctx, releaseID, digest, nowSeconds, nowSeconds)
	if err != nil {
		return PublishOutcomeUnsupported, fmt.Errorf("claiming release '%s': %w", releaseID, err)
	}

	if !won {
		tookOver, outcome, takeoverErr := takeOverContestedRelease(ctx, publisher, releaseID, digest, now)
		if takeoverErr != nil {
			return outcome, takeoverErr
		}
		if !tookOver {
			l.Internal("Release publish deferred to another node",
				logger_domain.String(logFieldRelease, releaseID),
				logger_domain.String("outcome", outcome.String()))
			return outcome, nil
		}
	}

	published, err := publishLayers(ctx, overlay, releaseID, artefacts, publishHeartbeatInterval, config.replicateBlob, time.Now)
	if err != nil {
		return PublishOutcomePublished, err
	}

	if err := publisher.MarkReleasePublished(ctx, releaseID, nowSeconds, nowSeconds); err != nil {
		return PublishOutcomePublished, fmt.Errorf("marking release '%s' published: %w", releaseID, err)
	}

	l.Internal("Published release layers into the shared overlay",
		logger_domain.String(logFieldRelease, releaseID),
		logger_domain.Int("artefacts", len(artefacts)),
		logger_domain.Int("newly_inserted", published))
	return PublishOutcomePublished, nil
}

// takeOverContestedRelease attempts to recover a publish whose owner died mid-flight.
// When the contested lease is still publishing and its heartbeat is older than
// publishingTakeoverTTL, it deletes the stale lease and makes exactly one re-claim
// attempt.
//
// Takes publisher (ReleasePublisher) which owns the lease rows.
// Takes releaseID (string) and digest (string) which identify the release and its
// payload.
// Takes now (time.Time) which is the claim time.
//
// Returns bool which reports whether this caller now owns the publish.
// Returns PublishOutcome which is the contested lease's terminal outcome for a caller
// that does not take over, so the caller can report it without re-reading the lease.
// Returns error when a lease read, delete, or claim fails.
func takeOverContestedRelease(
	ctx context.Context,
	publisher ReleasePublisher,
	releaseID, digest string,
	now time.Time,
) (bool, PublishOutcome, error) {
	outcome, lease, err := inspectContestedRelease(ctx, publisher, releaseID, digest)
	if err != nil {
		return false, outcome, err
	}
	if outcome != PublishOutcomeInProgress || lease.State != ReleaseStatePublishing {
		return false, outcome, nil
	}

	staleCutoffSeconds := now.Add(-publishingTakeoverTTL).Unix()
	if lease.HeartbeatAt >= staleCutoffSeconds {
		return false, outcome, nil
	}

	ctx, l := logger_domain.From(ctx, log)
	l.Warn("Taking over a stale publishing lease whose owner stopped heartbeating",
		logger_domain.String(logFieldRelease, releaseID),
		logger_domain.Int64("lease_heartbeat_at", lease.HeartbeatAt))
	if err := publisher.DeleteStalePublishingLease(ctx, releaseID, staleCutoffSeconds); err != nil {
		return false, outcome, fmt.Errorf("deleting stale publishing lease for '%s': %w", releaseID, err)
	}

	nowSeconds := now.Unix()
	won, err := publisher.ClaimRelease(ctx, releaseID, digest, nowSeconds, nowSeconds)
	if err != nil {
		return false, outcome, fmt.Errorf("re-claiming release '%s' after takeover: %w", releaseID, err)
	}
	return won, outcome, nil
}

// publishLayers writes every artefact as an immutable release layer and increments blob
// reference counts only for layers actually inserted.
//
// When a replicator is supplied, each variant's bytes are copied into the shared store
// BEFORE its record is written, so a crash mid-publish leaves harmless unreferenced bytes
// rather than a record without bytes. Each artefact is cloned and stamped with the
// release id before insertion: seed artefacts always carry an empty ReleaseID (the
// payload format does not serialise it), and an unstamped insert would land on the
// runtime layer key and silently break multi-release coexistence.
//
// A long publish advances its own lease heartbeat every beatEvery between layers, so a
// takeover-eligible staleness can only mean the publisher genuinely died. A zero
// beatEvery heartbeats after every layer, which exists for tests.
//
// Takes overlay (MetadataStore) which owns the layer rows, blob reference counts and
// lease; it must also implement ReleasePublisher, which the caller has already
// established.
// Takes releaseID (string) which stamps every layer.
// Takes artefacts ([]*registry_dto.ArtefactMeta) which are the seed artefacts to publish.
// Takes beatEvery (time.Duration) which paces the mid-publish heartbeats.
// Takes replicateBlob (BlobReplicator) which copies a variant's bytes ahead of its
// record, or nil to skip byte replication.
// Takes nowFunc (func() time.Time) which reads the current time, so heartbeat pacing is
// testable.
//
// Returns int which is the number of layers newly inserted.
// Returns error when an insert, reference increment, or heartbeat fails.
func publishLayers(
	ctx context.Context,
	overlay MetadataStore,
	releaseID string,
	artefacts []*registry_dto.ArtefactMeta,
	beatEvery time.Duration,
	replicateBlob BlobReplicator,
	nowFunc func() time.Time,
) (int, error) {
	publisher, ok := overlay.(ReleasePublisher)
	if !ok {
		return 0, fmt.Errorf("overlay for publishing release '%s' does not implement ReleasePublisher", releaseID)
	}

	published := 0
	lastBeat := nowFunc()
	for _, artefact := range artefacts {
		if artefact == nil {
			continue
		}

		layer := artefact.Clone()
		layer.ReleaseID = releaseID
		if err := replicateLayerBlobs(ctx, replicateBlob, layer); err != nil {
			return published, err
		}

		inserted, err := insertLayerWithReferences(ctx, overlay, layer, releaseID)
		if err != nil {
			return published, err
		}
		if inserted {
			published++
		}

		beat, err := maybeHeartbeat(ctx, publisher, releaseID, beatEvery, lastBeat, nowFunc)
		if err != nil {
			return published, err
		}
		lastBeat = beat
	}
	return published, nil
}

// replicateLayerBlobs copies every variant's bytes into the shared store ahead of the
// record, honouring the write-ahead rule so a crash mid-publish leaves harmless
// unreferenced bytes rather than a record whose bytes a foreign node cannot fetch. It is
// a no-op when no replicator is configured.
//
// Takes replicateBlob (BlobReplicator) which copies one variant's bytes, or nil.
// Takes layer (*registry_dto.ArtefactMeta) whose variants are replicated.
//
// Returns error when a variant's bytes cannot be replicated.
func replicateLayerBlobs(ctx context.Context, replicateBlob BlobReplicator, layer *registry_dto.ArtefactMeta) error {
	if replicateBlob == nil {
		return nil
	}
	for i := range layer.ActualVariants {
		if err := replicateBlob(ctx, &layer.ActualVariants[i]); err != nil {
			return fmt.Errorf("replicating bytes for variant '%s' of artefact '%s': %w",
				layer.ActualVariants[i].VariantID, layer.ID, err)
		}
	}
	return nil
}

// insertLayerWithReferences writes one immutable release layer and increments its blob
// reference counts in a single transaction.
//
// The single transaction means a crash can never leave a committed layer whose blobs were
// never counted: either both the row and its references land, or neither does. The
// reference increments run only when the row was newly inserted, so racing nodes never
// double-count.
//
// Takes overlay (MetadataStore) which provides the transaction and the reference counts.
// Takes layer (*registry_dto.ArtefactMeta) which is the stamped layer to write.
// Takes releaseID (string) which the layer is published under, for error context.
//
// Returns bool which reports whether a row was newly inserted.
// Returns error when the transaction, insert, or reference increment fails.
func insertLayerWithReferences(ctx context.Context, overlay MetadataStore, layer *registry_dto.ArtefactMeta, releaseID string) (bool, error) {
	inserted := false
	err := overlay.RunAtomic(ctx, func(ctx context.Context, transactionStore MetadataStore) error {
		publisher, ok := transactionStore.(ReleasePublisher)
		if !ok {
			return fmt.Errorf("transaction store for publishing release '%s' lost the ReleasePublisher capability", releaseID)
		}
		var insertErr error
		inserted, insertErr = publisher.InsertArtefactLayerIfAbsent(ctx, layer)
		if insertErr != nil {
			return fmt.Errorf("publishing artefact layer '%s' for release '%s': %w", layer.ID, releaseID, insertErr)
		}
		if !inserted {
			return nil
		}
		if refErr := incrementLayerRefCounts(ctx, transactionStore, layer); refErr != nil {
			return fmt.Errorf("referencing blobs for published artefact '%s': %w", layer.ID, refErr)
		}
		return nil
	})
	return inserted, err
}

// maybeHeartbeat advances the release lease heartbeat when at least beatEvery has elapsed
// since the last beat, so a long publish keeps its lease fresh and a takeover-eligible
// staleness can only mean the publisher genuinely died. A zero beatEvery beats after
// every layer, for tests.
//
// Takes publisher (ReleasePublisher) which owns the lease row.
// Takes releaseID (string) which identifies the release.
// Takes beatEvery (time.Duration) which is the minimum interval between beats.
// Takes lastBeat (time.Time) which is when the heartbeat last advanced.
// Takes nowFunc (func() time.Time) which reads the current time.
//
// Returns time.Time which is the timestamp of the last beat after this call.
// Returns error when the heartbeat update fails.
func maybeHeartbeat(
	ctx context.Context,
	publisher ReleasePublisher,
	releaseID string,
	beatEvery time.Duration,
	lastBeat time.Time,
	nowFunc func() time.Time,
) (time.Time, error) {
	if nowFunc().Sub(lastBeat) < beatEvery {
		return lastBeat, nil
	}
	if err := publisher.HeartbeatRelease(ctx, releaseID, nowFunc().Unix()); err != nil {
		return lastBeat, fmt.Errorf("heartbeating release '%s' during publish: %w", releaseID, err)
	}
	return nowFunc(), nil
}

// inspectContestedRelease reads the lease another node already wrote and classifies what
// a caller that lost the publish claim faces. It never signals the caller to proceed
// itself; a stale-publishing takeover is a separate decision made by
// takeOverContestedRelease.
//
// A published lease carrying the same digest means the publish is already complete, so
// the caller does nothing (PublishOutcomeAlreadyPublished). A published lease carrying a
// different digest means two different binaries claimed one release id, which is
// unrecoverable and returned as an ErrReleaseDigestConflict-wrapped error. Any other
// state (still publishing, or a lease that vanished) is reported as
// PublishOutcomeInProgress.
//
// Takes publisher (ReleasePublisher) which owns the lease rows.
// Takes releaseID (string) which identifies the contested release.
// Takes digest (string) which is this build's release payload digest.
//
// Returns PublishOutcome which is the terminal outcome for the caller.
// Returns ReleaseLease which is the contested lease when it exists, for staleness
// inspection.
// Returns error when a digest conflict is detected or the lease read fails.
func inspectContestedRelease(
	ctx context.Context,
	publisher ReleasePublisher,
	releaseID, digest string,
) (PublishOutcome, ReleaseLease, error) {
	lease, exists, err := publisher.GetRelease(ctx, releaseID)
	if err != nil {
		return PublishOutcomeUnsupported, ReleaseLease{}, fmt.Errorf("reading contested release '%s': %w", releaseID, err)
	}
	if !exists {
		return PublishOutcomeInProgress, ReleaseLease{}, nil
	}
	if lease.State == ReleaseStatePublished {
		if lease.PublishDigest != digest {
			return PublishOutcomeUnsupported, lease, fmt.Errorf(
				"release '%s' already published a different payload (existing digest %q, this build %q): two builds must not share a release id: %w",
				releaseID, lease.PublishDigest, digest, ErrReleaseDigestConflict)
		}
		return PublishOutcomeAlreadyPublished, lease, nil
	}
	return PublishOutcomeInProgress, lease, nil
}

// incrementLayerRefCounts increments blob reference counts for every variant of a freshly
// published artefact layer, so the shared overlay's ref-count-gated GC never reclaims a
// blob a live release still serves. It reuses the same per-variant reference logic as a
// runtime AddVariant, so build and runtime blobs are accounted identically.
//
// Takes store (MetadataStore) which owns the blob reference counts.
// Takes artefact (*registry_dto.ArtefactMeta) which holds the variants to reference.
//
// Returns error when a reference count update fails.
func incrementLayerRefCounts(ctx context.Context, store MetadataStore, artefact *registry_dto.ArtefactMeta) error {
	for i := range artefact.ActualVariants {
		variant := &artefact.ActualVariants[i]
		if err := incrementVariantRefCounts(ctx, store, variant); err != nil {
			return fmt.Errorf("referencing variant '%s': %w", variant.VariantID, err)
		}
	}
	return nil
}

// RetireRelease removes every artefact layer of a release from a shared overlay,
// decrements the blob references those layers held, emits garbage-collection hints for
// blobs that reached zero, and drops the release's lease. It is a no-op when the overlay
// does not provide ReleasePublisher.
//
// The whole retire runs in one transaction: the layer delete returns the deleted layers
// in a single statement, so of two racing reapers exactly one observes the rows and
// decrements, and a failure rolls everything back for the reaper to retry. Blobs shared
// with another live release keep a positive reference count and are never hinted for
// collection.
//
// Retiring a release can never break a node still running it: that node serves its own
// records from its own in-memory base and its own bytes from its own binary, and never
// consulted the overlay for its own content. A premature retire only degrades
// cross-release serving until the release is republished, which the dropped lease
// permits.
//
// Takes overlay (MetadataStore) which is the shared writable overlay.
// Takes releaseID (string) which identifies the release to retire.
//
// Returns error when the reclaim, reference decrements, hint emission, or lease drop
// fails.
func RetireRelease(ctx context.Context, overlay MetadataStore, releaseID string) error {
	if _, ok := overlay.(ReleasePublisher); !ok {
		return nil
	}
	if releaseID == "" {
		return errors.New("cannot retire a release with an empty release id")
	}

	ctx, l := logger_domain.From(ctx, log)
	err := overlay.RunAtomic(ctx, func(ctx context.Context, transactionStore MetadataStore) error {
		return retireReleaseInTransaction(ctx, transactionStore, releaseID)
	})
	if err != nil {
		return err
	}
	l.Internal("Retired release layers, blob references, and lease from the shared overlay",
		logger_domain.String(logFieldRelease, releaseID))
	return nil
}

// retireReleaseInTransaction performs the retire steps against the transaction-scoped
// store: reclaim (delete and return) the release's layers, decrement every reclaimed
// variant's blob and chunk references collecting hints for blobs that reached zero, drop
// the lease, and record the hints.
//
// Takes transactionStore (MetadataStore) which is the transaction-scoped store; it must
// also implement ReleasePublisher (the transaction clone of a layer-aware store does).
// Takes releaseID (string) which identifies the release to retire.
//
// Returns error when any step fails, rolling the transaction back.
func retireReleaseInTransaction(ctx context.Context, transactionStore MetadataStore, releaseID string) error {
	publisher, ok := transactionStore.(ReleasePublisher)
	if !ok {
		return fmt.Errorf("transaction store for retiring release '%s' lost the ReleasePublisher capability", releaseID)
	}

	layers, err := publisher.ReclaimArtefactLayersForRelease(ctx, releaseID)
	if err != nil {
		return fmt.Errorf("retiring release '%s' layers: %w", releaseID, err)
	}

	hints := make([]registry_dto.GCHint, 0)
	for _, layer := range layers {
		layerHints, hintErr := collectVariantGCHints(ctx, transactionStore, layer.ActualVariants)
		if hintErr != nil {
			return fmt.Errorf("dereferencing blobs for retired artefact '%s': %w", layer.ID, hintErr)
		}
		hints = append(hints, layerHints...)
	}

	if err := publisher.DeleteReleaseLease(ctx, releaseID); err != nil {
		return fmt.Errorf("dropping release '%s' lease: %w", releaseID, err)
	}

	if len(hints) > 0 {
		if err := transactionStore.AtomicUpdate(ctx, []registry_dto.AtomicAction{{
			Type:    registry_dto.ActionTypeAddGCHints,
			GCHints: hints,
		}}); err != nil {
			return fmt.Errorf("recording garbage hints for retired release '%s': %w", releaseID, err)
		}
	}
	return nil
}

// HeartbeatRelease advances a release's heartbeat, telling a reaper the release is still
// live. It is a no-op when the overlay does not provide ReleasePublisher.
//
// The heartbeat only ever moves forward: the store advances it when the new value is more
// recent, so an out-of-order heartbeat cannot rewind a fresher one written by another
// node of the same release.
//
// Takes overlay (MetadataStore) which is the shared writable overlay.
// Takes releaseID (string) which identifies the release to heartbeat.
// Takes now (time.Time) which is the heartbeat time.
//
// Returns error when the heartbeat update fails.
func HeartbeatRelease(ctx context.Context, overlay MetadataStore, releaseID string, now time.Time) error {
	publisher, ok := overlay.(ReleasePublisher)
	if !ok || releaseID == "" {
		return nil
	}
	if err := publisher.HeartbeatRelease(ctx, releaseID, now.Unix()); err != nil {
		return fmt.Errorf("heartbeating release '%s': %w", releaseID, err)
	}
	return nil
}

// ReapExpiredReleases retires every published release whose most recent heartbeat is
// older than the time-to-live, excluding the caller's own release so a node never reaps
// what it is serving. It is a no-op when the overlay does not provide ReleasePublisher.
//
// It errs generous: a release is retired only when its heartbeat predates now minus the
// TTL, so a briefly-unhealthy node keeps its release, and the cost of waiting is a few
// megabytes of unreferenced records held for the TTL rather than an incident from reaping
// a live release.
//
// Takes overlay (MetadataStore) which is the shared writable overlay.
// Takes ownRelease (string) which is excluded from reaping.
// Takes ttl (time.Duration) which is how stale a heartbeat must be before its release is
// reaped.
// Takes now (time.Time) which is the current time.
//
// Returns int which is the number of releases retired.
// Returns error when listing or retiring a release fails.
func ReapExpiredReleases(
	ctx context.Context,
	overlay MetadataStore,
	ownRelease string,
	ttl time.Duration,
	now time.Time,
) (int, error) {
	publisher, ok := overlay.(ReleasePublisher)
	if !ok {
		return 0, nil
	}

	ctx, l := logger_domain.From(ctx, log)
	cutoffSeconds := now.Add(-ttl).Unix()
	expired, err := publisher.ListExpiredReleases(ctx, cutoffSeconds, ownRelease)
	if err != nil {
		return 0, fmt.Errorf("listing expired releases: %w", err)
	}

	retired := 0
	for _, releaseID := range expired {
		if err := RetireRelease(ctx, overlay, releaseID); err != nil {
			return retired, err
		}
		retired++
	}
	if retired > 0 {
		l.Internal("Reaped expired releases", logger_domain.Int("retired", retired))
	}
	return retired, nil
}
