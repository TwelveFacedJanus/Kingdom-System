import "../style.css";

export default function OurSponsors() {
    return (
        <div className="third-block--dark_theme">
            <div className="corner-line bottom-right-1"></div>
            <div className="corner-line bottom-right-2"></div>
            <div className="corner-line bottom-right-3"></div>
            <div className="corner-line bottom-right-4"></div>
            <div className="corner-line top-left-1"></div>
            <div className="corner-line top-left-2"></div>
            <div className="corner-line top-left-3"></div>
            <div className="our-sponsors--dark_theme" style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                textAlign: 'center',
                padding: '2rem 0'
                }}>
                <p className="our-sponsors--tile--dark_theme">End-to-End Encryption</p>
                <p className="our-sponsors--description--dark_theme">All your data in safe.</p>
            </div>
      </div>
    );
}