//go:build js && wasm

package config

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"slices"
	"syscall/js"
	"time"
)

// AddEventListener is a function that adds an event listener to the document.
func AddEventListener(event string, listener any) {
	document.Call("addEventListener", event, listener)
}

// AddEventListenerToCanvas is a function that adds an event listener to the document.
func AddEventListenerToCanvas(event string, listener any) {
	canvasObject.Call("addEventListener", event, listener)
}

// CanvasBoundingBox returns the bounding box of the document.
// getBoundingClientRect forces the browser to flush pending layout, and the
// result is read once per enemy, bullet and input event, so it is cached until
// invalidateCanvasBoundingBox reports that the canvas may have moved.
func CanvasBoundingBox() dimensions {
	canvasBoxMutex.RLock()
	dim, valid := canvasBox, canvasBoxValid
	canvasBoxMutex.RUnlock()

	if valid {
		return dim
	}

	box := canvasObject.Call("getBoundingClientRect")

	dim = dimensions{
		BoxLeft:        box.Get("left").Float(),
		BoxTop:         box.Get("top").Float(),
		BoxRight:       box.Get("right").Float(),
		BoxBottom:      box.Get("bottom").Float(),
		BoxWidth:       box.Get("width").Float(),
		BoxHeight:      box.Get("height").Float(),
		OriginalWidth:  originalWidth,
		OriginalHeight: originalHeight,
	}

	dim.ScaleWidth = dim.BoxWidth / dim.OriginalWidth
	dim.ScaleHeight = dim.BoxHeight / dim.OriginalHeight

	canvasBoxMutex.Lock()
	canvasBox, canvasBoxValid = dim, true
	canvasBoxMutex.Unlock()

	return dim
}

// invalidateCanvasBoundingBox discards the cached canvas geometry so that the
// next CanvasBoundingBox call measures the document again.
func invalidateCanvasBoundingBox() {
	canvasBoxMutex.Lock()
	canvasBoxValid = false
	canvasBoxMutex.Unlock()
}

// ClearBackground is a function that clears the invisible document.
func ClearBackground() {
	invisibleCtx.Call("clearRect", 0, 0, invisibleCanvas.Get("width"), invisibleCanvas.Get("height"))
}

// ClearCanvas is a function that clears the document.
func ClearCanvas() {
	canvasObjectContext.Call("clearRect", 0, 0, canvasObject.Get("width"), canvasObject.Get("height"))
}

// ConvertArrayToSlice is a function that converts an array to a slice.
func ConvertArrayToSlice(array js.Value) []any {
	length := array.Length()
	result := make([]any, length)
	for i := 0; i < length; i++ {
		element := array.Index(i)
		switch element.Type() {
		case js.TypeObject:
			if element.InstanceOf(js.Global().Get("Array")) {
				result[i] = ConvertArrayToSlice(element)
			} else {
				result[i] = ConvertObjectToMap(element)
			}
		case js.TypeString:
			result[i] = element.String()
		case js.TypeNumber:
			result[i] = element.Float()
		case js.TypeBoolean:
			result[i] = element.Bool()
		case js.TypeNull, js.TypeUndefined:
			result[i] = nil
		default:
			result[i] = element
		}
	}
	return result
}

// ConvertObjectToMap is a function that converts an object to a map.
func ConvertObjectToMap(obj js.Value) map[string]any {
	result := make(map[string]any)
	keys := GlobalGet("Object").Call("keys", obj)
	for i := 0; i < keys.Length(); i++ {
		key := keys.Index(i).String()
		value := obj.Get(key)

		switch value.Type() {
		case js.TypeObject:
			if value.InstanceOf(GlobalGet("Array")) {
				result[key] = ConvertArrayToSlice(value)
			} else {
				result[key] = ConvertObjectToMap(value)
			}
		case js.TypeString:
			result[key] = value.String()
		case js.TypeNumber:
			result[key] = value.Float()
		case js.TypeBoolean:
			result[key] = value.Bool()
		case js.TypeNull, js.TypeUndefined:
			result[key] = nil
		default:
			result[key] = value
		}
	}

	return result
}

// Getenv is a function that returns the value of the environment variable key.
func Getenv(key string) string {
	got := GlobalGet(goEnv).Get(key)
	if !got.Truthy() {
		return ""
	}

	return got.String()
}

// GetScores is a function that returns the scores.
func GetScores(top int) (scores []score) {
	scoreBoardMutex.RLock()
	defer scoreBoardMutex.RUnlock()

	for i := 0; i < top && i < len(scoreBoard); i++ {
		scores = append(scores, scoreBoard[i])
	}

	return
}

// GlobalCall is a function that calls the global function name with the specified arguments.
func GlobalCall(name string, args ...any) js.Value {
	return js.Global().Call(name, args...)
}

// GlobalGet is a function that returns the global value of key.
func GlobalGet(key string) js.Value {
	return js.Global().Get(key)
}

// GlobalSet is a function that sets the global value of key to value.
func GlobalSet(key string, value any) {
	switch value := value.(type) {
	case js.Value:
		js.Global().Set(key, value)
	default:
		js.Global().Set(key, js.ValueOf(value))
	}
}

// IsPlaying is a function that returns true if the audio track is playing.
func IsPlaying(name string) bool {
	audioPlayersMutex.RLock()
	defer audioPlayersMutex.RUnlock()

	player, playerOk := audioPlayers[name]
	return playerOk && player.source.Truthy()
}

// IsTouchDevice is a function that returns true if the device is a touch device.
func IsTouchDevice() bool {
	navigator := GlobalGet("navigator")
	switch {
	case window.Call("hasOwnProperty", "ontouchstart").Bool():
		return true

	case navigator.Truthy():
		if maxTouchPoints := navigator.Get("maxTouchPoints"); maxTouchPoints.Truthy() && maxTouchPoints.Int() > 0 {
			return true
		}

		if msMaxTouchPoints := navigator.Get("msMaxTouchPoints"); msMaxTouchPoints.Truthy() && msMaxTouchPoints.Int() > 0 {
			return true
		}

	}

	return false
}

// LoadAudio is a function that loads an audio file.
func LoadAudio(name string) ([]byte, error) {
	protocol := windowLocation.Get("protocol").String()
	hostname := windowLocation.Get("hostname").String()
	port := windowLocation.Get("port").String()

	url := fmt.Sprintf("%s//%s:%s/audio/%s", protocol, hostname, port, name)
	if port == "" {
		url = fmt.Sprintf("%s//%s/audio/%s", protocol, hostname, name)
	}

	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return raw, nil
}

// Log is a function that logs a message.
func Log(msg string) {
	console.Call("log", msg)
}

// LogError is a function that logs an error.
func LogError(err error) {
	if err != nil {
		console.Call("error", err.Error())
	}
}

// MakeObject is a function that returns a new object with the specified key-value pairs.
func MakeObject(m map[string]any) js.Value {
	obj := NewInstance("Object")
	for key, value := range m {
		obj.Set(key, value)
	}
	return obj
}

// NewInstance is a function that returns a new instance of the type with the specified arguments.
func NewInstance(typ string, args ...any) js.Value {
	return GlobalGet(typ).New(args...)
}

// ensureAudioContext lazily (re)creates the audio context and reports whether
// it can be used.
func ensureAudioContext() bool {
	if audioCtx.Truthy() {
		return true
	}

	audioCtx = getAudioContext()
	if !audioCtx.Truthy() {
		LogError(fmt.Errorf("failed to initialize audio context"))
		return false
	}

	return true
}

// audioBuffer resolves the decoded AudioBuffer of the named track and hands it
// to onReady, or hands it js.Null if the track could not be made available.
// A track is fetched and decoded at most once for the lifetime of the page:
// decoding is orders of magnitude more expensive than starting a buffer source,
// and requests arriving while a decode is in flight are queued rather than
// starting a decode of their own.
func audioBuffer(name string, onReady func(buffer js.Value)) {
	if !ensureAudioContext() {
		onReady(js.Null())
		return
	}

	audioTracksMutex.Lock()
	track, trackOk := audioTracks[name]
	if !trackOk {
		track = &audioTrack{}
		audioTracks[name] = track
	}

	if track.buffer.Truthy() {
		buffer := track.buffer
		audioTracksMutex.Unlock()
		onReady(buffer)
		return
	}

	track.waiting = append(track.waiting, onReady)
	if track.loading {
		audioTracksMutex.Unlock()
		return
	}

	track.loading = true
	audioTracksMutex.Unlock()

	go func() {
		raw, err := LoadAudio(name)
		if err != nil {
			LogError(fmt.Errorf("failed to load audio track %s: %w", name, err))
			settleAudioTrack(track, js.Null())
			return
		}

		buffer := NewInstance("Uint8Array", len(raw))
		js.CopyBytesToJS(buffer, raw)

		// decodeAudioData detaches the underlying ArrayBuffer, so the copy above
		// cannot be reused - which is fine, it is made once per track.
		var then, catch js.Func
		then = js.FuncOf(func(_ js.Value, p []js.Value) any {
			then.Release()
			catch.Release()
			settleAudioTrack(track, p[0])
			return nil
		})
		catch = js.FuncOf(func(_ js.Value, p []js.Value) any {
			then.Release()
			catch.Release()
			LogError(fmt.Errorf("failed to decode audio track %s: %s\n%s",
				name, p[0].Get("message").String(), p[0].Get("stack").String()))
			settleAudioTrack(track, js.Null())
			return nil
		})

		audioCtx.Call("decodeAudioData", buffer.Get("buffer")).Call("then", then).Call("catch", catch)
	}()
}

// settleAudioTrack publishes the outcome of a decode and drains the callbacks
// that queued behind it. Failures are reported too, so that callers waiting on
// the track do not stay blocked forever.
func settleAudioTrack(track *audioTrack, buffer js.Value) {
	audioTracksMutex.Lock()
	track.loading = false
	if buffer.Truthy() {
		track.buffer = buffer
	}
	waiting := track.waiting
	track.waiting = nil
	audioTracksMutex.Unlock()

	for _, onReady := range waiting {
		onReady(buffer)
	}
}

// preloadAudio fetches and decodes every shipped audio track in the background,
// so that the frame which first triggers a sound does not pay for it.
func preloadAudio() {
	if !*Config.Control.AudioEnabled {
		return
	}

	for _, name := range audioTrackNames {
		audioBuffer(name, func(js.Value) {})
	}
}

// PlayAudio is a function that plays an audio track.
func PlayAudio(name string, loop bool) {
	if !*Config.Control.AudioEnabled || !ensureAudioContext() {
		return
	}

	audioPlayersMutex.Lock()
	player, playerOk := audioPlayers[name]
	if !playerOk {
		player = &audioPlayer{source: js.Null()}
		audioPlayers[name] = player
	}

	// A track is never layered over itself; a request arriving while the
	// previous one still plays (or is waiting for its buffer) is dropped.
	if busy := player.source.Truthy() || player.starting; busy {
		audioPlayersMutex.Unlock()

		if Config.Control.Debug.Get() {
			Log(fmt.Sprintf("Audio source already playing: %s", name))
		}
		return
	}

	player.loop, player.starting = loop, true
	audioPlayersMutex.Unlock()

	// Resolves immediately once the track has been decoded, which after the
	// initial preload is every time.
	audioBuffer(name, func(buffer js.Value) { startAudioSource(name, player, buffer) })
}

// startAudioSource wires a decoded buffer up to the destination and starts it.
// Creating a buffer source is cheap, so one is made per playback and dropped
// afterwards, as the Web Audio API requires.
func startAudioSource(name string, player *audioPlayer, buffer js.Value) {
	audioPlayersMutex.Lock()
	defer audioPlayersMutex.Unlock()

	// StopAudio may have cancelled the playback while the buffer was decoding.
	if !player.starting {
		return
	}
	player.starting = false

	if !buffer.Truthy() || player.source.Truthy() {
		return
	}

	// Browsers hand out a suspended context until the first user gesture.
	// Resuming a running context is a no-op.
	if audioCtx.Get("state").String() == "suspended" {
		audioCtx.Call("resume")
	}

	source := audioCtx.Call("createBufferSource")
	source.Set("buffer", buffer)
	// Let the Web Audio API repeat the track: restarting it from the ended
	// callback leaves an audible gap and crosses into Go on every repetition.
	source.Set("loop", player.loop)
	source.Call("connect", audioCtx.Get("destination"))

	if !player.endedCallback.Truthy() {
		player.endedCallback = js.FuncOf(func(_ js.Value, _ []js.Value) any {
			audioPlayersMutex.Lock()
			player.source = js.Null()
			player.offset = 0
			audioPlayersMutex.Unlock()

			return nil
		})
	}
	source.Call("addEventListener", "ended", player.endedCallback)

	// Resume where StopAudio left off, wrapping around for looped tracks.
	offset := player.offset
	if duration := buffer.Get("duration").Float(); duration > 0 {
		offset = math.Mod(offset, duration)
	}

	player.source = source
	player.startedAt = audioCtx.Get("currentTime").Float() - offset

	if Config.Control.Debug.Get() {
		Log(fmt.Sprintf("Playing audio source: %s", name))
	}

	source.Call("start", 0, offset)
}

// SaveScores is a function that saves the score board persistently.
func SaveScores() {
	scoreBoardMutex.RLock()
	serialized, err := json.Marshal(scoreBoard)
	scoreBoardMutex.RUnlock()

	if err != nil {
		LogError(fmt.Errorf("failed to serialize score board: %v", err))
		return
	}

	// Save the score board
	SendMessage(Execute(Config.MessageBox.Messages.WaitForScoreBoardUpdate), false, false)
	scoreBoardMutex.Lock()

	GlobalCall("fetch", "scores.db", MakeObject(map[string]any{
		"method":  http.MethodPut,
		"headers": MakeObject(map[string]any{"Content-Type": "application/json"}),
		"body":    string(serialized),
	})).Call("then", js.FuncOf(func(_ js.Value, p []js.Value) any {
		defer scoreBoardMutex.Unlock()

		if !p[0].Get("ok").Bool() {
			return p[0].Call("text").Call("then", js.FuncOf(func(_ js.Value, p []js.Value) any {
				LogError(fmt.Errorf("server responded with error: %s", p[0].String()))
				return nil
			}))
		}

		// Send success message
		SendMessage(Execute(Config.MessageBox.Messages.ScoreBoardUpdated), false, false)
		return nil
	})).Call("catch", js.FuncOf(func(_ js.Value, p []js.Value) any {
		defer scoreBoardMutex.Unlock()

		LogError(fmt.Errorf("failed to save score board: %s", p[0].String()))
		return nil
	}))
}

// SendInfoMessage sends a message to the message box.
func SendMessage(msg string, reset, event logEvent) {
	channel := event.Channel()
	channelBtn := event.ChannelButton()

	if reset {
		// Reset the content, keeping only the new message
		channel.Set("innerHTML", msg)
	} else {
		// Create a DocumentFragment
		fragment := document.Call("createDocumentFragment")

		// Create a temporary container element
		div := document.Call("createElement", "div")
		div.Set("innerHTML", msg)

		// Move all children from the temporary container to the fragment
		fragment.Call("appendChild", div)

		// Append the fragment to the channel, minimizing DOM manipulation
		channel.Call("appendChild", fragment)

		// Limit the number of messages in the DOM
		for channel.Get("children").Length() > Config.MessageBox.ChannelBufferSize {
			channel.Call("removeChild", channel.Get("firstChild"))
		}
	}

	// Scroll to the beginning of the newly added message
	channel.Get("lastChild").Call("scrollIntoView", MakeObject(map[string]any{
		"block":    "start",
		"behavior": "smooth",
	}))

	if channelBtn.Get("classList").Call("contains", tabActiveClass).Bool() {
		return // Already active, nothing to do
	}

	// Flash the channel.
	// The listener is created once per channel in setupMessageBoxInterface and
	// re-registered here: allocating a js.FuncOf per message leaked one entry of
	// the global callback registry for every enemy hit, and only Release frees
	// them. Registering the same function twice is a no-op, so the "once" option
	// still leaves exactly one listener attached.
	channelBtn.Get("classList").Call("add", tabFlashClass)
	channelBtn.Call("addEventListener", "animationend", flashEndCallbacks[event], flashEndOptions)
}

// SendMessageThrottled sends a message to the message box with a cooldown.
// The cooldown is tracked per topic so that a chatty message cannot suppress an
// unrelated one, which a single shared timestamp used to do.
func SendMessageThrottled(topic string, msg string, reset, event logEvent, cooldown time.Duration) {
	lastLogSentMutex.Lock()
	last, seen := lastLogSentTime[topic]
	if seen && time.Since(last) < cooldown {
		lastLogSentMutex.Unlock()
		return
	}

	lastLogSentTime[topic] = time.Now()
	lastLogSentMutex.Unlock()

	SendMessage(msg, reset, event)
}

// Setenv is a function that sets the environment variable key to value.
func Setenv(key, value string) {
	environ := GlobalGet(goEnv)
	environ.Set(key, value)
	GlobalSet(goEnv, environ)
	invalidateEnvCache()
}

// SetScore is a function that sets the score.
func SetScore(name string, newScore int) (rank int) {
	scoreBoardMutex.Lock()

	// Update the score board
	var exists bool
	for i, s := range scoreBoard {
		if s.Name == name {
			if newScore <= s.Score {
				scoreBoardMutex.Unlock()
				return len(scoreBoard) + 1
			}

			scoreBoard[i].Score = newScore
			exists = true
			break
		}
	}

	// Add the score if it does not exist
	if !exists {
		scoreBoard = append(scoreBoard, score{Name: name, Score: newScore})
	}

	// Sort the score board
	slices.SortStableFunc(scoreBoard, scoreBoardSortFunc)
	scoreBoardMutex.Unlock()

	// Calculate the rank of the new score
	scoreBoardMutex.RLock()
	for i, s := range scoreBoard {
		if s.Name == name && s.Score == newScore {
			rank = i + 1
			break
		}
	}
	scoreBoardMutex.RUnlock()

	// Save the score board asynchronously
	go SaveScores()

	return
}

// StopAudio is a function that stops an audio track.
func StopAudio(name string) {
	audioPlayersMutex.Lock()
	defer audioPlayersMutex.Unlock()

	player, playerOk := audioPlayers[name]
	if !playerOk {
		return
	}

	// Cancel a playback that is still waiting for its buffer, so that it does
	// not start after the caller asked for silence.
	player.starting = false

	if !player.source.Truthy() {
		return
	}

	if Config.Control.Debug.Get() {
		Log(fmt.Sprintf("Stopping audio source: %s", name))
	}

	// Remember how far into the track playback got, so that a later PlayAudio
	// resumes rather than restarts. The offset has to be relative to the start
	// of the source: audioCtx.currentTime is measured from the creation of the
	// context, and passing it straight to start() silenced the track as soon as
	// the context had been alive longer than the track lasts.
	player.offset = audioCtx.Get("currentTime").Float() - player.startedAt

	// The listener is removed first so that the pending "ended" event of the
	// stopped source cannot clear the state of a playback started after it.
	player.source.Call("removeEventListener", "ended", player.endedCallback)
	player.source.Call("stop")
	player.source = js.Null()
}

// StopAudioSources is a function that stops all audio sources that match the selector.
func StopAudioSources(selector func(name string) bool) {
	audioPlayersMutex.RLock()

	var stopped []string
	for name, player := range audioPlayers {
		if selector(name) && (player.source.Truthy() || player.starting) {
			stopped = append(stopped, name)
		}
	}

	audioPlayersMutex.RUnlock()

	for _, name := range stopped {
		StopAudio(name)
	}

	if Config.Control.Debug.Get() {
		Log(fmt.Sprintf("Stopped audio sources: %v", stopped))
	}
}

// ThrowError is a function that throws an error.
func ThrowError(err error) {
	if err != nil {
		js.Global().Call("eval", fmt.Sprintf("throw new Error('%s')", err.Error()))
	}
}

// Unsetenv is a function that unsets the environment variable key.
func Unsetenv(key string) {
	environ := GlobalGet(goEnv)
	environ.Delete(key)
	GlobalSet(goEnv, environ)
	invalidateEnvCache()
}

// UpdateHUD refreshes the on-screen display.
// Only the values that changed are written back, because touching the DOM on
// every frame for numbers that change a few times a minute would undo the point
// of having a heads-up display at all.
func UpdateHUD(state HUD) {
	if state == lastHUD {
		return
	}

	if state.Score != lastHUD.Score {
		hudScoreSpan.Set("textContent", state.Score)
	}

	if state.Level != lastHUD.Level {
		hudLevelSpan.Set("textContent", state.Level)
	}

	if state.Cannons != lastHUD.Cannons {
		hudCannonsSpan.Set("textContent", state.Cannons)
	}

	if state.Experience != lastHUD.Experience {
		hudExperienceBar.Get("style").Set("width", fmt.Sprintf("%.1f%%", state.Experience*100))
	}

	if state.ShieldCharge != lastHUD.ShieldCharge || state.ShieldCapacity != lastHUD.ShieldCapacity {
		var pips string
		for i := 0; i < state.ShieldCapacity; i++ {
			if i < state.ShieldCharge {
				pips += `<i class="charged"></i>`
				continue
			}

			pips += "<i></i>"
		}
		hudShieldPips.Set("innerHTML", pips)
	}

	lastHUD = state
}

// UpdateFPS is a function that updates the frames per second.
func UpdateFPS(fps float64) {
	fpsDiv.Set("innerHTML", fmt.Sprintf(fpsDiv.Call("getAttribute", "data-format").String(), fps))
}
