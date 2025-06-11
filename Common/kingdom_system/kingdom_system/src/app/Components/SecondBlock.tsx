import Image from "next/image";
import '../style.css';

export default function SecondBlock() {
    return (
        <div className="second-block--dark-theme">
        <div className="corner-line bottom-right-1"></div>
        <div className="corner-line bottom-right-2"></div>
        <div className="corner-line bottom-right-3"></div>
        <div className="corner-line bottom-right-4"></div>
        <div className="corner-line top-left-1"></div>
        <div className="corner-line top-left-2"></div>
        
        <div className="features-block--dark-theme">
          <div className="features-block--content--dark-theme">
            <div className="features-block--content-title-with-icon">
              <Image src="/fast.svg" alt="int" width={20} height={20} />
              <p className="features-block--content--title--dark-theme">Fast</p>
            </div>
            <p className="features-block--content--description--dark-theme">Written from scratch for more efficently leverage multiple CPU cores and your GPU.</p>
          </div>
          <div className="features-block--content--dark-theme">
          <div className="features-block--content-title-with-icon">
              <Image src="/safe.svg" alt="int" width={20} height={20} />
              <p className="features-block--content--title--dark-theme">Safe</p>
            </div>
            <p className="features-block--content--description--dark-theme">Written from scratch for more efficently leverage multiple CPU cores and your GPU.</p>
          </div>
          <div className="features-block--content--dark-theme">
            <div className="features-block--content-title-with-icon">
              <Image src="/intelligent.svg" alt="int" width={20} height={20} />
              <p className="features-block--content--title--dark-theme">Intelligent</p>
            </div>
            <p className="features-block--content--description--dark-theme">Written from scratch for more efficently leverage multiple CPU cores and your GPU.</p>
          </div>
        </div>

        {/* YouTube Video Player Section */}
        <div className="youtube-player-container" style={{ 
          width: '100%', 
          maxWidth: '1000px', 
          margin: '2rem auto',
          padding: '1rem'
        }}>
          <iframe width="100%" height="600" src="https://rutube.ru/play/embed/0fd630b54e05f904b6862783161e7b74/" frameBorder="0" allow="clipboard-write; autoplay"></iframe>
        </div>
        
      </div>
    );
}