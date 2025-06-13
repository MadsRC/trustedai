// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { animate, motion, useMotionValue } from "framer-motion";
import { useEffect } from "react";

interface TypewriterProps {
  text: string;
  className?: string;
  duration?: number;
}

export default function Typewriter({
  text,
  className = "",
  duration = 1.5,
}: TypewriterProps) {
  const displayText = useMotionValue("");

  useEffect(() => {
    const animation = animate(0, text.length, {
      duration,
      ease: "linear",
      onUpdate: (latest) => {
        displayText.set(text.slice(0, Math.ceil(latest)));
      },
    });

    return () => animation.stop();
  }, [text, displayText, duration]);

  return (
    <div className={`inline-flex items-center ${className}`}>
      <motion.span className="font-mono">{displayText}</motion.span>
      <motion.div
        className="w-0.5 h-6 bg-current ml-1"
        animate={{
          opacity: [1, 1, 0, 0],
        }}
        transition={{
          duration: 1,
          repeat: Infinity,
          times: [0, 0.5, 0.5, 1],
        }}
      />
    </div>
  );
}
