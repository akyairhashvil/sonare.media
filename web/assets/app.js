        // --- 10) SECURITY: Basic Sanitization Helper ---
        function escapeHtml(text) {
            if (!text) return text;
            return text
                .replace(/&/g, "&amp;")
                .replace(/</g, "&lt;")
                .replace(/>/g, "&gt;")
                .replace(/"/g, "&quot;")
                .replace(/'/g, "&#039;");
        }

        // --- DATA & CONFIG ---
        const SELECTION = {
            'Calm': { name: 'First Light Drift', desc: 'Minimal ambient textures with high focus retention. Ideal for bookshops, spas, and high-end galleries.' },
            'Warm': { name: 'Analog Hearth', desc: 'Acoustic warmth, soft jazz guitar, and neo-soul textures. Welcoming and intimate.' },
            'Modern': { name: 'Throughput Pulse', desc: 'Downtempo electronic and minimal house. Clean, efficient, forward-thinking.' },
            'Upbeat': { name: 'Kinetic Retail', desc: 'Nu-disco and indie dance influence. Keeps energy high without aggression.' },
            'Premium': { name: 'Velvet Lounge', desc: 'Cinematic minimal and modern jazz. Sophisticated background for luxury items.' },
            'Playful': { name: 'Bright Motif', desc: 'Light funk, upbeat lo-fi, and clean rhythmic pop instrumentals. Smile-inducing.' }
        };

        // --- NEW PRICING LOGIC ---
        // Set these to numbers (e.g., 850, 75, 35) when you are ready to reveal prices.
        const PRICING = {
            base_fee: 3500,  // e.g. 850
            per_hour: 250,   // e.g. 75
            per_store: 500   // e.g. 200
        };

        const fmt = (n) => {
            if (n === null || n === undefined) return "TBD";
            return new Intl.NumberFormat('en-US', { style: "currency", currency: "USD" }).format(n);
        };

        // --- LEGAL CONTENT REPOSITORY ---
        const LEGAL_DOCS = {
            'privacy': `
                <h3>01 // Data Collection Protocol</h3>
                <p>We collect only the essential data points required to execute a Sound Audit: Name, Email Address, Business Identity, and Operational Metrics. This data is transmitted securely and is never sold.</p>
                <h3>02 // Data Retention</h3>
                <p>Client data is retained for the duration of the project lifecycle. Upon completion, sensitive operational data is purged unless a maintenance retainer is active.</p>
            `,
            'terms': `
                <h3>01 // Asset Sovereignty</h3>
                <p>Upon full payment, the Client receives a perpetual, non-exclusive license to utilize the curated playlist architecture. You own the files for playback purposes.</p>
                <h3>02 // Licensing Responsibility</h3>
                <p>Sonare provides the "Architectural Curation." The Client is responsible for maintaining necessary public performance licenses (PROs) required by local jurisdiction.</p>
            `,
            'cookies': `
                <h3>01 // Local Storage</h3>
                <p>We use "Local Storage" to remember session preferences (palette, calculator). This is strictly functional.</p>
            `
        };

        // --- STATE ---
        let userAnswers = {};
        
        // --- GLOBAL STATUS UPDATE HELPER ---
        function updateGlobalStatus(msg) {
            const el = document.getElementById('global-status');
            el.style.opacity = '0';
            setTimeout(() => {
                el.innerText = msg;
                el.style.opacity = '0.6';
            }, 300);
        }
        
        // --- MOBILE MENU LOGIC ---
        function toggleMobileMenu() {
            const menu = document.getElementById('mobile-menu');
            const toggle = document.querySelector('.mobile-toggle');
            menu.classList.toggle('active');
            toggle.classList.toggle('open');
        }

        // --- NAV HIGHLIGHT LOGIC ---
        window.addEventListener('scroll', () => {
            let current = '';
            const sections = document.querySelectorAll('section');
            const scrollY = window.scrollY;

            sections.forEach(section => {
                const sectionTop = section.offsetTop;
                if (scrollY >= (sectionTop - 200)) {
                    current = section.getAttribute('id');
                }
            });

            document.querySelectorAll('.nav-links a').forEach(a => {
                a.classList.remove('active');
                if (a.getAttribute('href').includes(current)) {
                    a.classList.add('active');
                }
            });
        });

        // --- GRAPH SWITCHING LOGIC ---
        function switchGraph(view) {
            // 2) ACCESSIBILITY: Update aria-pressed
            const buttons = document.querySelectorAll('.graph-controls .toggle-pill');
            buttons.forEach(btn => {
                btn.classList.remove('active');
                btn.setAttribute('aria-pressed', 'false');
            });

            const activeBtn = document.getElementById('btn-graph-' + view);
            if(activeBtn) {
                activeBtn.classList.add('active');
                activeBtn.setAttribute('aria-pressed', 'true');
            }

            // Update Graph Layers
            const layers = document.querySelectorAll('.graph-layer');
            layers.forEach(l => l.classList.add('graph-hidden'));
            
            document.getElementById('view-' + view).classList.remove('graph-hidden');

            // Update Label
            const label = document.getElementById('y-axis-label');
            if (view === 'energy') label.innerText = "ENERGY / INTENSITY";
            if (view === 'vocals') label.innerText = "LYRICAL DENSITY";
            if (view === 'beacons') label.innerText = "IDENTITY EVENTS";
            
            // 6) STATUS: Update Global Status
            updateGlobalStatus(`GRAPH VIEW: ${view.toUpperCase()}`);
        }

        // --- QUESTIONNAIRE LOGIC ---
        function selectOption(e, category, value) {
            userAnswers[category] = value;
            updateGlobalStatus(`PALETTE: ${category.toUpperCase()} SET TO ${value.toUpperCase()}`);
            
            // Navigate
            const steps = ['q1', 'q2', 'q3', 'q4', 'q-result'];
            let currentIdx = -1;
            
            if (category === 'vibe') currentIdx = 0;
            if (category === 'energy') currentIdx = 1;
            if (category === 'texture') currentIdx = 2;
            if (category === 'vocals') currentIdx = 3;

            if (currentIdx > -1 && currentIdx < 4) {
                const currentStep = document.getElementById(steps[currentIdx]);
                
                // Show sticky nav on mobile when quiz starts
                document.body.classList.add('quiz-active');
                document.getElementById('quiz-progress').innerText = `STEP 0${currentIdx + 2}/05`;
                
                const buttons = currentStep.querySelectorAll('button');
                buttons.forEach(b => b.classList.remove('selected'));
                e.target.classList.add('selected');

                setTimeout(() => {
                    currentStep.classList.remove('active');
                    document.getElementById(steps[currentIdx + 1]).classList.add('active');
                    if (currentIdx === 3) generateResult();
                }, 300);
            }
        }

        function generateResult() {
            let paletteKey = userAnswers.vibe || 'Modern'; 
            if (paletteKey === 'Calm' && userAnswers.texture === 'Organic') paletteKey = 'Warm';
            if (paletteKey === 'Upbeat' && userAnswers.texture === 'Electronic') paletteKey = 'Modern';

            const result = SELECTION[paletteKey] || SELECTION['Modern'];
            
            // Hide sticky nav on result
            document.body.classList.remove('quiz-active');
            updateGlobalStatus(`RESULT GENERATED: ${result.name.toUpperCase()}`);

            // 10) SECURITY: Escape output before rendering
            document.getElementById('res-name').innerText = result.name; 
            document.getElementById('res-desc').innerText = `Based on your need for ${escapeHtml(userAnswers.vibe)} vibes and ${escapeHtml(userAnswers.energy)} energy.`;
            
            // Context updates for Form
            document.getElementById('form-palette').value = `${result.name} (${paletteKey})`;
            document.getElementById('ctx-palette').innerText = `${result.name}`;

            // --- BUILD PREVIEW CARDS (NEW) ---
            const phases = [
                { key: "open", name: "OPEN", title: "First Light" },
                { key: "peak", name: "PEAK", title: "Core Flow" },
                { key: "offpeak", name: "OFF-PEAK", title: "Drift State" },
                { key: "close", name: "CLOSE", title: "Last Call" }
            ];

            const gridHTML = phases.map(phase => `
                <div class="preview-card">
                    <div>
                        <div class="mono" style="font-size:0.7rem; color:var(--accent-cyan); margin-bottom:0.25rem;">${phase.name}</div>
                        <h3 style="font-size:1rem; margin-top:0; margin-bottom:0;">${phase.title}</h3>
                        <div class="wave-skeleton" data-track="${phase.key}" aria-hidden="true"></div>
                    </div>
                    <div>
                        <button class="audio-btn" data-track="${phase.key}" data-src="" disabled aria-disabled="true">
                            <span class="audio-label">▶ Preview</span>
                            <span class="mono audio-meta" style="font-size:0.65rem; opacity:0.6;">(SOON)</span>
                        </button>
                    </div>
                </div>
            `).join('');

            // --- BRAND BEACON (FULL-WIDTH ROW) ---
            // A single identity track preview per kit.
            const beaconHTML = `
    <div class="preview-card is-wide beacon-card">
        <div class="beacon-meta">
            <div class="mono" style="font-size:0.7rem; color:var(--accent-cyan); margin-bottom:0.25rem;">BRAND BEACON</div>
            <h3 style="font-size:1rem; margin-top:0; margin-bottom:0;">Identity Track</h3>
            <p style="margin:0.5rem 0 0; font-size:0.85rem; color:var(--fg-secondary); max-width:none;">
                A single, signature song used as the source motif for transition resets in full deployments.
            </p>
        </div>
        <div class="wave-skeleton" data-track="beacon" aria-hidden="true"></div>
        <div class="beacon-cta">
            <button class="audio-btn" data-track="beacon" data-src="" disabled aria-disabled="true">
                <span class="audio-label">▶ Preview</span>
                <span class="mono audio-meta" style="font-size:0.65rem; opacity:0.6;">(SOON)</span>
            </button>
        </div>
    </div>
`;


            const previewRoot = document.getElementById('res-preview-grid');
            previewRoot.innerHTML = gridHTML + beaconHTML;

            // Mount custom preview playback controls.
            // Backend hook: implement window.SonarePreviewKit.onRequestSources(ctx) or call setSources(...) later.
            if (window.SonarePreviewKit && typeof window.SonarePreviewKit.mount === 'function') {
                window.SonarePreviewKit.mount(previewRoot, {
                    paletteKey: paletteKey,
                    kitName: result.name,
                    answers: { ...userAnswers }
                });
            }
        }

        function resetQuiz() {
            userAnswers = {};
            if (window.SonarePreviewKit && typeof window.SonarePreviewKit.stop === 'function') {
                window.SonarePreviewKit.stop();
            }
            document.querySelectorAll('.quiz-step').forEach(el => el.classList.remove('active'));
            document.querySelectorAll('.option-btn').forEach(el => el.classList.remove('selected'));
            document.getElementById('q1').classList.add('active');
            document.body.classList.remove('quiz-active');
        }


        // --- PREVIEW PLAYBACK SYSTEM (CUSTOM UI; BACKEND-READY) ---
        // This page intentionally avoids native <audio controls>. We provide our own UI:
        // - Progress follows playback position (drives .wave-skeleton via CSS var --progress).
        // - Play button toggles to Pause while playing.
        // Backend integration hooks:
        // 1) Implement: window.SonarePreviewKit.onRequestSources = async (ctx) => ({ open, peak, offpeak, close, beacon })
        // 2) Or call: window.SonarePreviewKit.setSources({ open, peak, offpeak, close, beacon }) after mount().
        (function () {
            const TRACK_KEYS = ["open", "peak", "offpeak", "close", "beacon"];

            function clamp(n, min, max) { return Math.min(max, Math.max(min, n)); }

            function fmtTime(seconds) {
                const s = Math.max(0, Math.floor(seconds || 0));
                const m = Math.floor(s / 60);
                const r = s % 60;
                return String(m).padStart(2, "0") + ":" + String(r).padStart(2, "0");
            }

            class HtmlAudioEngine {
                constructor() {
                    this.audio = new Audio();
                    this.audio.preload = "metadata";
                    // Backend may serve signed URLs; crossOrigin helps avoid surprises for simple previews.
                    this.audio.crossOrigin = "anonymous";
                }
                load(url) {
                    if (!url) return;
                    if (this.audio.src !== url) this.audio.src = url;
                }
                play() { return this.audio.play(); }
                pause() { this.audio.pause(); }
                stop() {
                    this.audio.pause();
                    try { this.audio.currentTime = 0; } catch (e) {}
                }
                get currentTime() { return this.audio.currentTime || 0; }
                set currentTime(t) { try { this.audio.currentTime = t; } catch (e) {} }
                get duration() { return Number.isFinite(this.audio.duration) ? this.audio.duration : 0; }
                get paused() { return this.audio.paused; }
                on(evt, fn) { this.audio.addEventListener(evt, fn); }
            }

            const engine = new HtmlAudioEngine();

            const state = {
                root: null,
                ctx: null,
                els: new Map(),     // key -> { btn, wave, labelEl, metaEl }
                currentKey: null,
                rafId: null,
                pendingSeekPct: null
            };

            function getEl(key) { return state.els.get(key) || null; }

            function setWaveProgress(waveEl, pct) {
                if (!waveEl) return;
                const normalized = Math.round(clamp(pct, 0, 100) * 100) / 100;
                waveEl.style.setProperty("--progress", String(normalized));
            }

            function setBtnMode(key, mode) {
                const el = getEl(key);
                if (!el || !el.btn) return;

                const btn = el.btn;
                const label = el.labelEl;
                const meta = el.metaEl;

                if (mode === "disabled") {
                    btn.disabled = true;
                    btn.setAttribute("aria-disabled", "true");
                    btn.setAttribute("aria-pressed", "false");
                    if (label) label.textContent = "▶ Preview";
                    if (meta) meta.textContent = "(SOON)";
                    setWaveProgress(el.wave, 0);
                    if (el.wave) {
                        el.wave.classList.remove("is-seekable");
                        el.wave.classList.remove("track-active");
                    }
                    return;
                }

                // Enabled baseline
                btn.disabled = false;
                btn.setAttribute("aria-disabled", "false");

                if (mode === "ready") {
                    btn.setAttribute("aria-pressed", "false");
                    if (label) label.textContent = "▶ Preview";
                    if (meta) meta.textContent = "00:00 / --:--";
                    if (el.wave) {
                        el.wave.classList.add("is-seekable");
                        el.wave.classList.remove("track-active");
                    }
                    return;
                }

                if (mode === "paused") {
                    btn.setAttribute("aria-pressed", "false");
                    if (label) label.textContent = "▶ Play";
                    if (el.wave) el.wave.classList.add("track-active");
                    return;
                }

                if (mode === "playing") {
                    btn.setAttribute("aria-pressed", "true");
                    if (label) label.textContent = "|| Pause";
                    if (el.wave) el.wave.classList.add("track-active");
                    return;
                }
            }

            function setMetaTime(key, t, d) {
                const el = getEl(key);
                if (!el || !el.metaEl) return;
                const dur = d > 0 ? fmtTime(d) : "--:--";
                el.metaEl.textContent = `${fmtTime(t)} / ${dur}`;
            }

            function stopRaf() {
                if (state.rafId) cancelAnimationFrame(state.rafId);
                state.rafId = null;
            }

            function tick() {
                const key = state.currentKey;
                if (!key) return;

                const el = getEl(key);
                const d = engine.duration;
                const t = engine.currentTime;

                const pct = d > 0 ? (t / d) * 100 : 0;
                setWaveProgress(el ? el.wave : null, pct);
                setMetaTime(key, t, d);

                if (!engine.paused) state.rafId = requestAnimationFrame(tick);
            }

            async function startPlayback(key) {
                const el = getEl(key);
                if (!el || !el.btn || el.btn.disabled) return;

                const src = (el.btn.dataset && el.btn.dataset.src) ? el.btn.dataset.src : "";
                if (!src) return;

                // Stop any other track first
                if (state.currentKey && state.currentKey !== key) stop();

                state.currentKey = key;
                engine.load(src);

                // If user clicked on the waveform before playback, honor it after metadata loads
                if (state.pendingSeekPct != null && engine.duration > 0) {
                    engine.currentTime = clamp(state.pendingSeekPct, 0, 1) * engine.duration;
                    state.pendingSeekPct = null;
                }

                try {
                    await engine.play();
                    stopRaf();
                    setBtnMode(key, "playing");

                    // Keep other buttons in "ready"
                    TRACK_KEYS.forEach(k => { if (k !== key) setBtnMode(k, getEl(k)?.btn?.disabled ? "disabled" : "ready"); });

                    if (typeof updateGlobalStatus === "function") updateGlobalStatus(`PLAYING: ${key.toUpperCase()}`);
                    state.rafId = requestAnimationFrame(tick);
                } catch (e) {
                    console.warn("Playback blocked or failed:", e);
                    setBtnMode(key, "paused");
                }
            }

            function pausePlayback() {
                if (!state.currentKey) return;
                engine.pause();
                stopRaf();
                setBtnMode(state.currentKey, "paused");
                if (typeof updateGlobalStatus === "function") updateGlobalStatus("PAUSED");
            }

            function stop() {
                engine.stop();
                stopRaf();
                if (state.currentKey) {
                    const el = getEl(state.currentKey);
                    setWaveProgress(el ? el.wave : null, 0);
                    setBtnMode(state.currentKey, getEl(state.currentKey)?.btn?.disabled ? "disabled" : "ready");
                    setMetaTime(state.currentKey, 0, engine.duration);
                }
                state.currentKey = null;

                // Reset any non-disabled track to ready baseline
                TRACK_KEYS.forEach(k => {
                    const el = getEl(k);
                    if (!el || !el.btn) return;
                    if (el.btn.disabled) setBtnMode(k, "disabled");
                    else setBtnMode(k, "ready");
                    setWaveProgress(el.wave, 0);
                });
            }

            function toggle(key) {
                if (state.currentKey === key) {
                    if (engine.paused) startPlayback(key);
                    else pausePlayback();
                    return;
                }
                startPlayback(key);
            }

            function seekFromWaveClick(key, clientX) {
                const el = getEl(key);
                if (!el || !el.wave || !el.btn || el.btn.disabled) return;

                const rect = el.wave.getBoundingClientRect();
                const pct = rect.width > 0 ? clamp((clientX - rect.left) / rect.width, 0, 1) : 0;

                // If seeking the currently loaded track, set currentTime directly.
                if (state.currentKey === key && engine.duration > 0) {
                    engine.currentTime = pct * engine.duration;
                    // Update UI immediately even if paused.
                    setWaveProgress(el.wave, pct * 100);
                    setMetaTime(key, engine.currentTime, engine.duration);
                    return;
                }

                // Otherwise, remember the seek percent and start playback; we will apply on metadata load.
                state.pendingSeekPct = pct;
                startPlayback(key);
            }

            function attach(root) {
                state.root = root;
                state.els.clear();
                state.currentKey = null;
                stopRaf();

                // Map DOM elements by data-track
                TRACK_KEYS.forEach(key => {
                    const btn = root.querySelector(`.audio-btn[data-track="${key}"]`);
                    const wave = root.querySelector(`.wave-skeleton[data-track="${key}"]`);
                    if (!btn || !wave) return;

                    const labelEl = btn.querySelector(".audio-label");
                    const metaEl = btn.querySelector(".audio-meta");

                    state.els.set(key, { btn, wave, labelEl, metaEl });
                    setWaveProgress(wave, 0);

                    btn.addEventListener("click", () => toggle(key));
                    wave.addEventListener("click", (e) => seekFromWaveClick(key, e.clientX));
                });

                // Baseline UI state
                TRACK_KEYS.forEach(k => {
                    const el = getEl(k);
                    if (!el || !el.btn) return;
                    if (el.btn.disabled) setBtnMode(k, "disabled");
                    else setBtnMode(k, "ready");
                });
            }

            function setSources(sources) {
                if (!state.root) return;
                const src = sources || {};

                TRACK_KEYS.forEach(key => {
                    const el = getEl(key);
                    if (!el || !el.btn) return;

                    const url = src[key] || "";
                    if (url) {
                        el.btn.dataset.src = url;
                        el.btn.disabled = false;
                        el.btn.setAttribute("aria-disabled", "false");
                        setBtnMode(key, "ready");
                    } else {
                        el.btn.dataset.src = "";
                        setBtnMode(key, "disabled");
                    }
                });
            }

            async function maybeRequestSources(ctx) {
                // Backend may implement this to return URL manifest.
                // Expected return shape: { open, peak, offpeak, close, beacon }
                if (typeof api.onRequestSources === "function") {
                    try {
                        const sources = await api.onRequestSources(ctx);
                        if (sources && typeof sources === "object") setSources(sources);
                    } catch (e) {
                        console.warn("onRequestSources failed:", e);
                    }
                }
            }

            function mount(root, ctx) {
                stop();
                state.ctx = ctx || null;
                attach(root);
                maybeRequestSources(state.ctx);
            }

            // Keep UI in sync with engine lifecycle
            engine.on("loadedmetadata", () => {
                if (!state.currentKey) return;
                const key = state.currentKey;

                // Apply any pending seek now that duration is known
                if (state.pendingSeekPct != null && engine.duration > 0) {
                    engine.currentTime = clamp(state.pendingSeekPct, 0, 1) * engine.duration;
                    state.pendingSeekPct = null;
                }
                setMetaTime(key, engine.currentTime, engine.duration);
            });

            engine.on("ended", () => {
                if (!state.currentKey) return;
                const key = state.currentKey;
                const el = getEl(key);
                stopRaf();
                setWaveProgress(el ? el.wave : null, 0);
                setBtnMode(key, "ready");
                setMetaTime(key, 0, engine.duration);
                state.currentKey = null;
                if (typeof updateGlobalStatus === "function") updateGlobalStatus("ENDED");
            });

            const api = {
                mount,
                stop,
                setSources,
                getContext: () => state.ctx,
                // Backend hook: assign a function here that returns a manifest object.
                onRequestSources: null
            };

            window.SonarePreviewKit = api;
        })();

        // --- PREVIEW SOURCE BRIDGE (Go backend) ---
        if (window.SonarePreviewKit) {
            window.SonarePreviewKit.onRequestSources = async (ctx) => {
                const palette = (ctx && ctx.paletteKey) ? String(ctx.paletteKey).trim().toLowerCase() : "";
                if (!palette) return {};

                try {
                    const response = await fetch(`/api/preview-sources?palette=${encodeURIComponent(palette)}`, {
                        method: "GET",
                        headers: { "Accept": "application/json" }
                    });

                    if (!response.ok) {
                        throw new Error(`Failed to load preview sources: HTTP ${response.status}`);
                    }

                    const payload = await response.json();
                    const sources = (payload && typeof payload.sources === "object" && payload.sources) ? payload.sources : {};
                    const hasAny = Object.values(sources).some(Boolean);

                    if (typeof updateGlobalStatus === "function") {
                        updateGlobalStatus(hasAny ? `PREVIEWS READY: ${palette.toUpperCase()}` : `NO PREVIEWS FOUND: ${palette.toUpperCase()}`);
                    }

                    return sources;
                } catch (error) {
                    console.error("Preview source fetch failed:", error);
                    if (typeof updateGlobalStatus === "function") {
                        updateGlobalStatus("PREVIEW SOURCE LOAD FAILED");
                    }
                    return {};
                }
            };
        }


        // --- PRICING LOGIC (Refactored) ---
        function updatePricingUI() {
            const hours = parseInt(document.getElementById('hours-input').value);
            const stores = parseInt(document.getElementById('stores-input').value);

            // Update UI Labels
            document.getElementById('hours-display').innerText = hours;
            document.getElementById('stores-display').innerText = stores;
            
            // 6) STATUS: Context feedback
            // Debounce/throttle logic could be added here for production, simple text update for now
            // updateGlobalStatus(`CALCULATOR: ${hours}H / ${stores} LOC`);
            
            // Update Context Box in Contact Form
            document.getElementById('ctx-scale').innerText = `${hours}h / ${stores} Store(s)`;
            document.getElementById('form-hours').value = hours;
            document.getElementById('form-stores').value = stores;

            // Update Rates Table
            document.getElementById('priceBase').textContent = fmt(PRICING.base_fee);
            document.getElementById('pricePerHour').textContent = fmt(PRICING.per_hour);
            document.getElementById('pricePerStore').textContent = fmt(PRICING.per_store);

            // Calculate Estimate
            const anyTBD = [PRICING.base_fee, PRICING.per_hour, PRICING.per_store].some(v => v === null);
            const estEl = document.getElementById("priceEstimate");

            if (anyTBD) {
                estEl.textContent = "TBD";
                estEl.style.color = "var(--fg-secondary)";
                return;
            }

            const estimate = PRICING.base_fee + (hours * PRICING.per_hour) + (stores * PRICING.per_store);
            estEl.textContent = fmt(estimate);
            estEl.style.color = "var(--accent-cyan)";
        }

        // --- CONTACT LOGIC (HARDENED) ---
        document.getElementById('contact-form').addEventListener('submit', (e) => {
            e.preventDefault();
            const btn = e.target.querySelector('button[type="submit"]');
            const status = document.getElementById('form-status');
            
            // 5) CLIENT-SIDE VALIDATION: Prevent duplicates
            if (btn.disabled) return;
            btn.innerText = "Transmitting...";
            btn.disabled = true;

            // 4) INCLUDE MESSAGE IN PAYLOAD
            const data = {
                name: document.getElementById('name').value,
                email: document.getElementById('email').value,
                business: document.getElementById('business').value,
                system: document.getElementById('playback').value,
                message: document.getElementById('msg').value, // Added
                // Hidden Context fields
                palette: document.getElementById('form-palette').value || "Not generated",
                hours_est: document.getElementById('form-hours').value,
                store_count: document.getElementById('form-stores').value
            };

            fetch('/api/lead', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(data)
            })
            .then(response => {
                if (response.ok) {
                    status.innerText = "> TRANSMISSION SUCCESSFUL.";
                    status.style.color = "var(--accent-cyan)";
                    btn.innerText = "Sent";
                    updateGlobalStatus("TRANSMISSION CONFIRMED");
                    e.target.reset();
                } else {
                    throw new Error('Network response was not ok');
                }
            })
            .catch(error => {
                console.error('Error:', error);
                status.innerText = "> TRANSMISSION FAILED. TRY AGAIN.";
                status.style.color = "var(--accent-alert)";
                btn.innerText = "Retry";
                btn.disabled = false;
            });
        });

        // --- LEGAL & COOKIE LOGIC (HARDENED) ---
        
        function openLegal(type) {
            const overlay = document.getElementById('legal-overlay');
            const title = document.getElementById('legal-title');
            const content = document.getElementById('legal-content');
            
            if(type === 'privacy') title.innerText = "PRIVACY PROTOCOLS";
            if(type === 'terms') title.innerText = "TERMS OF USE";
            if(type === 'cookies') title.innerText = "COOKIE POLICY";
            
            content.innerHTML = LEGAL_DOCS[type];
            overlay.classList.add('open');
            
            // 3) ACCESSIBILITY: Focus management
            document.querySelector('.close-btn').focus();
        }

        function closeLegal() {
            document.getElementById('legal-overlay').classList.remove('open');
        }

        // 3) ACCESSIBILITY: Escape Key Support
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                closeLegal();
            }
        });
        
        // 3) ACCESSIBILITY: Click Outside Support
        document.getElementById('legal-overlay').addEventListener('click', (e) => {
            if (e.target === e.currentTarget) {
                closeLegal();
            }
        });

        // --- INIT ---
        document.addEventListener("DOMContentLoaded", () => {
            document.getElementById('hours-input').addEventListener('input', updatePricingUI);
            document.getElementById('stores-input').addEventListener('input', updatePricingUI);
            updatePricingUI();

            // Safe Cookie Banner logic
            const btnAccept = document.getElementById('btn-accept-cookies');
            if(btnAccept) {
                btnAccept.addEventListener('click', () => {
                    try {
                        localStorage.setItem('sonare_consent', 'true');
                    } catch (e) {
                        console.warn("Storage restricted.");
                    }
                    document.getElementById('cookie-banner').classList.remove('active');
                });
            }

            // Check Consent State
            let hasConsent = false;
            try {
                hasConsent = localStorage.getItem('sonare_consent');
            } catch(e) {}

            if (!hasConsent) {
                setTimeout(() => {
                    const banner = document.getElementById('cookie-banner');
                    if(banner) banner.classList.add('active');
                }, 1000);
            }
        });

