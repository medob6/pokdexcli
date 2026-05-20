import { useEffect, useMemo, useState } from "react";
import CardGrid from "./components/CardGrid.jsx";

const limit = 10;

const titleize = (value) =>
	value
		.split("-")
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(" ");

export default function App() {
	const [offset, setOffset] = useState(0);
	const [locations, setLocations] = useState([]);
	const [locationsCount, setLocationsCount] = useState(0);
	const [loadingLocations, setLoadingLocations] = useState(false);
	const [explore, setExplore] = useState({ area: "", pokemon: [] });
	const [encounter, setEncounter] = useState({ id: "", expiresAt: "" });
	const [timeLeft, setTimeLeft] = useState(0);
	const [catchName, setCatchName] = useState("");
	const [catchStatus, setCatchStatus] = useState(null);
	const [pokedex, setPokedex] = useState([]);
	const [gameState, setGameState] = useState({
		level: 1,
		xp: 0,
		next_level_xp: 100,
		streak: 0,
		daily: { explores: 0, catches: 0, bonus_claimed: false },
		missions: [],
		bonus_claimed: false,
	});
	const [error, setError] = useState("");

	const lastPage = useMemo(() => {
		if (!locationsCount) {
			return false;
		}
		return offset + limit >= locationsCount;
	}, [locationsCount, offset]);

	const loadLocations = async () => {
		setLoadingLocations(true);
		setError("");
		try {
			const res = await fetch(`/api/locations?offset=${offset}&limit=${limit}`);
			if (!res.ok) {
				throw new Error("Failed to load locations");
			}
			const data = await res.json();
			setLocations(data.results || []);
			setLocationsCount(data.count || 0);
		} catch (err) {
			setError(err.message || "Something went wrong.");
		} finally {
			setLoadingLocations(false);
		}
	};

	const loadPokedex = async () => {
		try {
			const res = await fetch("/api/pokedex");
			if (!res.ok) {
				throw new Error("Failed to load pokedex");
			}
			const data = await res.json();
			setPokedex(data.pokemon || []);
		} catch (err) {
			setError(err.message || "Something went wrong.");
		}
	};

	const loadState = async () => {
		try {
			const res = await fetch("/api/state");
			if (!res.ok) {
				throw new Error("Failed to load state");
			}
			const data = await res.json();
			setGameState(data);
		} catch (err) {
			setError(err.message || "Something went wrong.");
		}
	};

	useEffect(() => {
		loadLocations();
	}, [offset]);

	useEffect(() => {
		loadPokedex();
		loadState();
	}, []);

	useEffect(() => {
		if (!encounter.expiresAt) {
			setTimeLeft(0);
			return;
		}
		const update = () => {
			const diff = new Date(encounter.expiresAt).getTime() - Date.now();
			setTimeLeft(Math.max(0, Math.ceil(diff / 1000)));
		};
		update();
		const timer = setInterval(update, 1000);
		return () => clearInterval(timer);
	}, [encounter.expiresAt]);

	const handleExplore = async (index) => {
		setError("");
		try {
			const res = await fetch(`/api/explore?index=${index}&limit=${limit}`);
			if (!res.ok) {
				throw new Error("Failed to explore location");
			}
			const data = await res.json();
			setExplore({ area: data.area, pokemon: data.pokemon || [] });
			setEncounter({ id: data.encounter_id || "", expiresAt: data.expires_at || "" });
			if (data.state) {
				setGameState(data.state);
			}
		} catch (err) {
			setError(err.message || "Something went wrong.");
		}
	};

	const handleCatch = async (event) => {
		event.preventDefault();
		const trimmed = catchName.trim();
		if (!trimmed) {
			setCatchStatus({ ok: false, message: "Enter a pokemon name first." });
			return;
		}
		setCatchStatus(null);
		setError("");
		try {
			const res = await fetch("/api/catch", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ name: trimmed, encounter_id: encounter.id || "" }),
			});
			if (!res.ok) {
				throw new Error("Catch request failed");
			}
			const data = await res.json();
			if (data.caught) {
				setCatchStatus({
					ok: true,
					message: `Caught ${titleize(data.pokemon.name)}! Rarity ${data.rarity}.`,
				});
				setCatchName("");
				loadPokedex();
			} else {
				setCatchStatus({
					ok: false,
					message: `${titleize(data.pokemon.name)} escaped. (${Math.round(
						data.catch_chance * 100
					)}% odds)`,
				});
			}
			if (data.state) {
				setGameState(data.state);
			}
		} catch (err) {
			setCatchStatus({ ok: false, message: err.message || "Catch failed." });
		}
	};

	return (
		<div className="app">
			<header className="hero">
				<div className="hero-top">
					<div className="brand">
						<svg
							className="pokeball"
							viewBox="0 0 120 120"
							role="img"
							aria-label="Pokeball logo"
						>
							<circle cx="60" cy="60" r="56" className="pokeball-shell" />
							<path
								d="M8 60h104"
								className="pokeball-line"
							/>
							<circle cx="60" cy="60" r="18" className="pokeball-core" />
							<circle cx="60" cy="60" r="10" className="pokeball-center" />
						</svg>
						<div>
							<p className="eyebrow">Trainer Control Room</p>
							<h1>Pokedex Field Guide</h1>
						</div>
					</div>
					<div className="hero-actions">
						<div className="pill">Level {gameState.level}</div>
						<div className="pill">XP {gameState.xp} / {gameState.next_level_xp}</div>
						<div className="pill">Streak {gameState.streak}</div>
					</div>
				</div>
				<p className="hero-lede">
					Map the region, scan for wild encounters, and secure new catches. Daily
					missions reset every morning.
				</p>
				<div className="badges">
					<div className="badge">Go-powered game logic</div>
					<div className="badge">Timed encounters</div>
					<div className="badge">Daily missions</div>
				</div>
			</header>

			{error ? <div className="status fail">{error}</div> : null}

			<section className="layout">
				<div className="panel">
					<h2>Location Map</h2>
					<p>Pick a location, then run a sweep for nearby pokemon activity.</p>
					<div className="controls">
						<button
							className="secondary"
							onClick={() => setOffset(Math.max(0, offset - limit))}
							disabled={offset === 0}
						>
							Prev
						</button>
						<button
							className="secondary"
							onClick={() => setOffset(offset + limit)}
							disabled={lastPage}
						>
							Next
						</button>
					</div>
					<div className="location-list">
						{loadingLocations ? (
							<div className="status">Loading locations...</div>
						) : (
							locations.map((loc) => (
								<div className="location-item" key={`${loc.index}-${loc.name}`}>
									<div>
										{titleize(loc.name)}
										<span> #{loc.index}</span>
									</div>
									<button className="secondary" onClick={() => handleExplore(loc.index)}>
										Explore
									</button>
								</div>
							))
						)}
					</div>
				</div>

				<div className="panel">
					<h2>Explorer</h2>
					<p>
						{explore.area
							? `Wild activity in ${titleize(explore.area)}.`
							: "Explore a location to see who appears."}
					</p>
					{encounter.expiresAt ? (
						<div className={`timer ${timeLeft === 0 ? "expired" : ""}`}>
							{timeLeft === 0
								? "Encounter expired. Scan again."
								: `Encounter window: ${timeLeft}s`}
						</div>
					) : null}
					<CardGrid
						title="Encounters"
						subtitle="Tap a name to preload the catch input."
						items={explore.pokemon}
						onSelect={(name) => setCatchName(name)}
					/>
					<form className="catch-row" onSubmit={handleCatch}>
						<input
							type="text"
							placeholder="Try catching..."
							value={catchName}
							onChange={(event) => setCatchName(event.target.value)}
							disabled={encounter.expiresAt && timeLeft === 0}
						/>
						<button type="submit" disabled={encounter.expiresAt && timeLeft === 0}>
							Catch
						</button>
					</form>
					{catchStatus ? (
						<div className={`status ${catchStatus.ok ? "success" : "fail"}`}>
							{catchStatus.message}
						</div>
					) : null}
				</div>
			</section>

			<section className="panel" style={{ marginTop: "20px" }}>
				<h2>Your Pokedex</h2>
				<p>Captured pokemon show up here with their stats.</p>
				{pokedex.length === 0 ? (
					<div className="status">No pokemon caught yet.</div>
				) : (
					<div className="grid">
						{pokedex.map((p) => (
							<div className="card" key={p.name}>
								<h3>{titleize(p.name)}</h3>
								{p.sprite ? <img src={p.sprite} alt={p.name} /> : <small>No sprite</small>}
								<small>ID: {p.id}</small>
								<div>
									{p.stats.map((stat) => (
										<div key={stat.name}>
											<small>
												{stat.name}: {stat.base}
											</small>
										</div>
									))}
								</div>
							</div>
						))}
					</div>
				)}
			</section>

			<section className="panel" style={{ marginTop: "20px" }}>
				<h2>Mission Board</h2>
				<p>Daily goals that grant bonus XP when completed.</p>
				<div className="missions">
					{gameState.missions?.map((mission) => (
						<div key={mission.key} className={`mission ${mission.done ? "done" : ""}`}>
							<div>
								<strong>{mission.label}</strong>
								<div className="mission-progress">
									{mission.progress} / {mission.goal}
								</div>
							</div>
							<span>{mission.done ? "Complete" : "In progress"}</span>
						</div>
					))}
					<div className="mission bonus">
						<div>
							<strong>Daily bonus</strong>
							<div className="mission-progress">+50 XP</div>
						</div>
						<span>{gameState.bonus_claimed ? "Claimed" : "Locked"}</span>
					</div>
				</div>
			</section>

			<div className="footer">Powered by PokeAPI. Progress saved locally.</div>
		</div>
	);
}
