'use client';

import Image from "next/image";
import { MouseEvent } from "react";

interface PrimaryButtonProps {
    text: string;
    icon?: {
        src: string;
        alt: string;
        width?: number;
        height?: number;
    };
    onClick?: (event: MouseEvent<HTMLButtonElement>) => void;
    className?: string;
    disabled?: boolean;
    type?: 'button' | 'submit' | 'reset';
}

export default function PrimaryButton({
    text,
    icon,
    onClick,
    className = '',
    disabled = false,
    type = 'button'
}: PrimaryButtonProps) {
    return (
        <button 
            className={`primary-button--dark_theme ${className}`}
            onClick={onClick}
            disabled={disabled}
            type={type}
        >
            {icon && (
                <Image 
                    src={icon.src} 
                    alt={icon.alt} 
                    width={icon.width || 20} 
                    height={icon.height || 20} 
                />
            )}
            {text}
        </button>
    );
}