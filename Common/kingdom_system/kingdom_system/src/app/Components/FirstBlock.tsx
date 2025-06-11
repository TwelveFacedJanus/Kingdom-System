"use client";

import '../style.css';
import Image from "next/image";

import PrimaryButton from './PrimaryButton';

export default function FirstBlock() {
    return (
        <div className="first_block--dark_theme">
        <div className="corner-line bottom-right-1"></div>
        <div className="corner-line bottom-right-2"></div>
        <div className="corner-line bottom-right-3"></div>
        <div className="corner-line bottom-right-4"></div>
        <div className="corner-line top-left-1"></div>
        <div className="corner-line top-left-2"></div>
        
        <div className="text-container--dark_theme">
          <p className="slogan--dark_theme">The system for education</p>
          <p className="description--dark_theme">Kingdom-System a next-gen and high-performance education assistant.</p>
          <div className="blurred-circles">
            <div className="blurred-circle"></div>
            <div className="blurred-circle"></div>
            <div className="blurred-circle"></div>
          </div>
        </div>
        <div className="download_buttons--dark_theme">
          <PrimaryButton text="Download now" icon={{src: "/download.svg", alt: "Download"}} onClick={(e) => console.log('Clicked')}/>
          <button className="clone-source-button--dark_theme">Clone source</button>
        </div>
      </div>
    );
}