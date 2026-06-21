export function EmptyState({ title, description }) {
  return (
    <div className="empty-state">
      <div className="empty-state-icon" aria-hidden="true">
        +
      </div>
      <h2>{title}</h2>
      <p>{description}</p>
    </div>
  )
}
