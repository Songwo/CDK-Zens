import { useEffect, useState } from "react";

export default function ParticleBurst({ trigger }) {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (!trigger) {
      return undefined;
    }

    setVisible(true);
    const timer = window.setTimeout(() => setVisible(false), 2200);
    return () => window.clearTimeout(timer);
  }, [trigger]);

  if (!visible) {
    return null;
  }

  const particles = Array.from({ length: 36 }, (_, index) => {
    const angle = (Math.PI * 2 * index) / 36;
    const distance = 80 + (Math.random() * 60);
    return {
      id: `${trigger}-${index}`,
      dx: `${Math.cos(angle) * distance}px`,
      dy: `${Math.sin(angle) * distance}px`,
      delay: `${Math.random() * 0.4}s`,
      size: `${6 + Math.random() * 4}px`,
    };
  });

  return (
    <div className="pointer-events-none fixed inset-0 z-[60] overflow-hidden">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(244,180,0,0.08),transparent_50%)]" />
      {particles.map((particle) => (
        <span
          key={particle.id}
          className="gold-particle"
          style={{
            "--dx": particle.dx,
            "--dy": particle.dy,
            "--delay": particle.delay,
            "--size": particle.size,
          }}
        />
      ))}
    </div>
  );
}
