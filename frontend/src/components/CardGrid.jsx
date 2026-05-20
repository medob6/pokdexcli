const titleize = (value) =>
	value
		.split("-")
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(" ");

const formatChance = (value) => `${Math.round(value * 100)}%`;

export default function CardGrid({ title, subtitle, items, onSelect }) {
	return (
		<div>
			<div style={{ marginBottom: "12px" }}>
				<strong>{title}</strong>
				<div style={{ color: "var(--text-muted)", fontSize: "13px" }}>{subtitle}</div>
			</div>
			{items.length === 0 ? (
				<div className="status">No encounters yet.</div>
			) : (
				<div className="grid">
					{items.map((item) => {
						const name = typeof item === "string" ? item : item.name;
						const rarity = typeof item === "string" ? "unknown" : item.rarity;
						const chance = typeof item === "string" ? null : item.catchChance;
						return (
						<button
							key={name}
							type="button"
							className="card secondary"
							onClick={() => onSelect?.(name)}
						>
							<div className="card-header">
								<h3>{titleize(name)}</h3>
								<span className={`rarity ${rarity}`}>{rarity}</span>
							</div>
							<small>
								{chance ? `Catch chance ${formatChance(chance)}` : "Tap to prep catch"}
							</small>
						</button>
					);
					})}
				</div>
			)}
		</div>
	);
}
