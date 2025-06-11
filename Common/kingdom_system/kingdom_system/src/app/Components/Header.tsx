"use client";
import "../style.css";
import { useState } from "react";
import Image from "next/image";
import CommonButton from "./CommonButton";
export function Header() {
    const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

    const toggleMobileMenu = () => {
        setIsMobileMenuOpen(!isMobileMenuOpen);
    };

    return (
        <>
            <div className="header--dark_theme">
                <div className="left_side--dark_theme">
                    <p className="logo--dark_theme">KS</p>
                    <CommonButton text="Releases" />
                    <CommonButton text="Plugins" />
                    <CommonButton text="Docs" />
                    <CommonButton text="Blog" />
                    <CommonButton text="Pricing" />
                    <CommonButton text="Resources" />
                </div>
                <div className="right_side--dark_theme">
                    <button className="menu-toggle" onClick={toggleMobileMenu}>
                        <Image src="/menu.svg" alt="Menu" width={24} height={24} />
                    </button>
                    <CommonButton text="Account" />
                    <button className="download_button--dark-theme">
                        Download
                    </button>
                </div>
            </div>
            <div className={`mobile-menu ${isMobileMenuOpen ? 'active' : ''}`}>
                <button className="common_button--dark-theme">
                    Releases
                </button>
                <button className="common_button--dark-theme">
                    Plugins
                </button>
                <button className="common_button--dark-theme">
                    Docs
                </button>
                <button className="common_button--dark-theme">
                    Blog
                </button>
                <button className="common_button--dark-theme">
                    Pricing
                </button>
                <button className="common_button--dark-theme">
                    Resources
                </button>
            </div>
        </>
    );
}

