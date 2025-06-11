'use client';

import Image from "next/image";
import { MouseEvent } from "react";

interface CommonButtonProps {
    text: string;
    icon?: {
        src: string;
        alt: string;
        width?: number;
        height?: number;
    };
    onClick?: (event: MouseEvent<HTMLButtonElement>) => void;
    className?: string;
    type?: 'button' | 'submit' | 'reset';
}

export default function CommonButton({
    text,
    icon,
    onClick,
    className = '',
    type = 'button',
}: CommonButtonProps)
{
    return (
        <button
            className={`common_button--dark-theme ${className}`}
            onClick={onClick}
            type={type}
        >
            { icon && (
                <Image
                    src={icon.src}
                    alt={icon.alt}
                    width={icon.width || 12}
                    height={icon.height || 12}
                />
            )}
            {text}
        </button>
    );
}